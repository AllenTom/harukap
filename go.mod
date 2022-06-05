module github.com/allentom/harukap

go 1.16

require (
	github.com/ahmetb/go-linq/v3 v3.2.0
	github.com/allentom/haruka v0.0.0-20220527084807-cad00e6ff194
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-resty/resty/v2 v2.7.0
	github.com/gorilla/websocket v1.4.2
	github.com/kardianos/service v1.2.0
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mitchellh/mapstructure v1.4.3
	github.com/project-xpolaris/youplustoolkit v0.0.0-20220331083706-51df568cbf83
	github.com/rs/cors v1.8.2 // indirect
	github.com/rs/xid v1.3.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/afero v1.8.2 // indirect
	github.com/spf13/viper v1.10.1
	github.com/urfave/cli/v2 v2.3.0
	go.etcd.io/etcd/client/v3 v3.5.2
	go.opentelemetry.io/otel v1.1.0
	go.opentelemetry.io/otel/exporters/jaeger v1.1.0
	go.opentelemetry.io/otel/sdk v1.1.0
	go.opentelemetry.io/otel/trace v1.1.0
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
	google.golang.org/genproto v0.0.0-20220317150908-0efb43f6373e // indirect
	google.golang.org/grpc v1.45.0
	gopkg.in/ini.v1 v1.66.4 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	gorm.io/driver/mysql v1.1.3
	gorm.io/driver/sqlite v1.2.3
	gorm.io/gorm v1.22.0
)
