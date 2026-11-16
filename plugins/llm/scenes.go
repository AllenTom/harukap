package llm

import (
	"sort"
	"strings"
)

// 场景表：代码里每个 LLM 调用点都对应一个场景 id。
//
// 为什么要有这张表：调用点写死了 prompt，但"用哪个模型"是运维决策。
// 把 id 暴露出来，前端才能给每个场景单独选模型；不做这张表的话，用户只能
// 全局换模型，改一处小任务也会把大任务一起换掉。
//
// 档位（Tier）只用于"用户没选"时挑默认模型：
//
//	fast      —— 短文本、高并发（搜索改写、打标签）
//	balanced  —— 默认
//	quality   —— 长文本推理（摘要、识别）
type SceneSpec struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Service     string `json:"service"`
	Tier        string `json:"tier"`
}

// SceneInfo 是场景的运行时状态，供管理接口返回。
type SceneInfo struct {
	SceneSpec
	// Model 实际会用的模型（空表示没解析出来，调用会退回提供商默认）
	Model string `json:"model"`
	// Source 模型来源：scene | service | provider | tier | none
	Source string `json:"source"`
	// Overridden 用户是否显式给这个场景选了模型
	Overridden bool `json:"overridden"`
}

// builtinScenes 是全部内置场景，顺序即前端展示顺序（按服务分组）。
func builtinScenes() []SceneSpec {
	return []SceneSpec{
		{SceneVideoTitleResolve, "视频识别 · 标题解析", "把文件名/目录名清理成可检索的标题候选", "video", "balanced"},
		{SceneVideoSummary, "视频摘要", "生成影片简介", "video", "quality"},

		{ScenePhotoSearchRewrite, "图片搜索 · 语义改写", "把中文口语查询改写成 CLIP 友好的英文短语", "photo", "fast"},
		{ScenePhotoSearchRewriteSig, "图片搜索 · SigLIP 标签改写", "改写成 Danbooru 风格标签", "photo", "fast"},
		{ScenePhotoCaption, "图片描述生成", "为图片生成描述文本", "photo", "quality"},

		{SceneMusicSearchRewrite, "音乐搜索 · 语义改写", "把查询改写成描述情绪/流派/乐器的英文短语", "music", "fast"},
		{SceneMusicArtistScrapeJudge, "音乐刮削 · 候选裁决", "从 Deezer/网易云/MusicBrainz 的候选里判断哪位是这位艺人、每个字段取哪家", "music", "balanced"},
		{SceneMusicTrackScrapeJudge, "音乐刮削 · 曲目候选裁决", "从网易云/MusicBrainz/LRCLIB 的候选里判断哪一条是这首曲目", "music", "balanced"},

		{SceneComicTagging, "漫画打标签", "从文件夹名/标题里抽取画师、系列、题材等标签", "comic", "fast"},
		{SceneComicSummary, "漫画标题翻译", "把日文/英文标题翻译成中文", "comic", "balanced"},
	}
}

// BuiltinScenes 返回全部内置场景（副本，调用方可安全修改）。
func BuiltinScenes() []SceneSpec {
	src := builtinScenes()
	out := make([]SceneSpec, len(src))
	copy(out, src)
	return out
}

// lookupScene 按 id 找内置场景定义。
func lookupScene(id string) (SceneSpec, bool) {
	for _, s := range builtinScenes() {
		if s.ID == id {
			return s, true
		}
	}
	return SceneSpec{}, false
}

// pickModelByTier 从提供商声明的模型里挑一个符合档位的。
// 没有 tier 标注时按顺序退化：quality→最后一个，fast→第一个，其他→第一个。
func pickModelByTier(models []ModelSpec, tier string) string {
	if len(models) == 0 {
		return ""
	}
	for _, m := range models {
		if strings.EqualFold(m.Tier, tier) && m.ID != "" {
			return m.ID
		}
	}
	switch strings.ToLower(tier) {
	case "quality":
		return models[len(models)-1].ID
	default:
		return models[0].ID
	}
}

// resolveSceneModel 计算场景最终使用的模型，并返回来源。
// 解析顺序：scene.model → service.model → provider.default_model → 按档位挑 → 空
func resolveSceneModel(cfg *LLMConfig, specs map[string]*ProviderSpec, scene string) (string, string) {
	if cfg == nil {
		return "", "none"
	}
	if scene != "" && cfg.Scenes != nil {
		if sel, ok := cfg.Scenes[scene]; ok && sel != nil {
			if m := strings.TrimSpace(sel.Model); m != "" {
				return m, "scene"
			}
		}
	}
	if m := strings.TrimSpace(cfg.Model); m != "" {
		return m, "service"
	}
	spec, ok := specs[cfg.Default]
	if !ok || spec == nil {
		return "", "none"
	}
	if m := strings.TrimSpace(spec.DefaultModel); m != "" {
		return m, "provider"
	}
	tier := "balanced"
	if s, found := lookupScene(scene); found {
		tier = s.Tier
	}
	if m := pickModelByTier(spec.Models, tier); m != "" {
		return m, "tier"
	}
	return "", "none"
}

// DescribeScenes 返回全部场景的运行状态。
//
// 只返回内置场景：用户配了但代码里没人调的场景（拼错 id）显示出来只会误导人，
// 所以额外收集 cfg.Scenes 里"没有对应调用点"的 id 单独标记出来。
func DescribeScenes(cfg *LLMConfig, specs map[string]*ProviderSpec) []SceneInfo {
	out := make([]SceneInfo, 0, len(builtinScenes())+4)

	for _, s := range builtinScenes() {
		model, source := resolveSceneModel(cfg, specs, s.ID)
		overridden := false
		if cfg != nil && cfg.Scenes != nil {
			if sel, ok := cfg.Scenes[s.ID]; ok && sel != nil && strings.TrimSpace(sel.Model) != "" {
				overridden = true
			}
		}
		out = append(out, SceneInfo{SceneSpec: s, Model: model, Source: source, Overridden: overridden})
	}

	if cfg == nil || len(cfg.Scenes) == 0 {
		return out
	}
	known := make(map[string]struct{}, len(out))
	for _, s := range out {
		known[s.ID] = struct{}{}
	}
	orphans := make([]SceneInfo, 0)
	for id, sel := range cfg.Scenes {
		if _, ok := known[id]; ok || sel == nil {
			continue
		}
		orphans = append(orphans, SceneInfo{
			SceneSpec: SceneSpec{
				ID: id, Label: id, Description: "配置里存在，但没有代码调用该场景",
				Service: "unknown", Tier: "",
			},
			Model: strings.TrimSpace(sel.Model), Source: "scene", Overridden: true,
		})
	}
	sort.Slice(orphans, func(i, j int) bool { return orphans[i].ID < orphans[j].ID })
	return append(out, orphans...)
}
