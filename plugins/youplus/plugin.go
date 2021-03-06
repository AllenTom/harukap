package youplus

import (
	"context"
	"fmt"
	"github.com/allentom/harukap"
	"github.com/allentom/harukap/commons"
	util "github.com/allentom/harukap/utils"
	"github.com/project-xpolaris/youplustoolkit/youplus"
	entry "github.com/project-xpolaris/youplustoolkit/youplus/entity"
	youplustoolkitrpc "github.com/project-xpolaris/youplustoolkit/youplus/rpc"
)

type Plugin struct {
	RPCClient     *youplustoolkitrpc.YouPlusRPCClient
	Client        *youplus.Client
	Entity        *entry.EntityClient
	AuthUrl       string
	AuthFromToken func(token string) (commons.AuthUser, error)
}

func (p *Plugin) OnInit(e *harukap.HarukaAppEngine) error {
	// rpc
	logger := e.LoggerPlugin.Logger.NewScope("youplusplugin")
	logger.Info("init plugin")
	enableRPC := e.ConfigProvider.Manager.GetBool("youplus.enablerpc")
	if enableRPC {
		rpcAddr := e.ConfigProvider.Manager.GetString("youplus.rpc")
		p.RPCClient = youplustoolkitrpc.NewYouPlusRPCClient(rpcAddr)
		err := p.RPCClient.Init()
		if err != nil {
			return err
		}
		// entity
		enableEntity := e.ConfigProvider.Manager.GetBool("youplus.entity.enable")
		if enableEntity {
			name := e.ConfigProvider.Manager.GetString("youplus.entity.name")
			version := e.ConfigProvider.Manager.GetInt64("youplus.entity.version")
			p.Entity = entry.NewEntityClient(name, version, &entry.EntityExport{}, p.RPCClient)
			p.Entity.HeartbeatRate = 3000
			// register entity
			p.Entity.Register()
			// set entity export
			addrs, err := util.GetHostIpList()
			urls := make([]string, 0)
			for _, addr := range addrs {
				urls = append(urls, fmt.Sprintf("http://%s%s", addr, e.ConfigProvider.Manager.GetString("addr")))
			}
			if err != nil {
				logger.Fatal(err.Error())
			}
			err = p.Entity.UpdateExport(entry.EntityExport{Urls: urls, Extra: map[string]interface{}{}})
			if err != nil {
				logger.Fatal(err.Error())
			}

			err = p.Entity.StartHeartbeat(context.Background())
			if err != nil {
				return err
			}

		}
	}
	// http
	p.Client = youplus.NewClient()
	p.Client.Init(e.ConfigProvider.Manager.GetString("youplus.url"))
	return nil
}
func (p *Plugin) GetAuthInfo() (*commons.AuthInfo, error) {
	authInfo := &commons.AuthInfo{
		Type: commons.AuthTypeBase,
		Name: "YouPlus",
		Url:  p.AuthUrl,
	}
	return authInfo, nil
}

func (p *Plugin) AuthName() string {
	return "youplus"
}

func (p *Plugin) TokenTypeName() string {
	return "YouPlusService"
}
func (p *Plugin) GetAuthUserByToken(token string) (commons.AuthUser, error) {
	if p.AuthFromToken == nil {
		return nil, nil
	}
	return p.AuthFromToken(token)
}
