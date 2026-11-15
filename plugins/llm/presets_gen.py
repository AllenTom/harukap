#!/usr/bin/env python3
"""生成 harukap/plugins/llm/presets.go（提供商预设目录）。

用法：
    curl -s https://models.dev/api.json -o /tmp/modelsdev.json
    python3 presets_gen.py && gofmt -w presets.go

为什么要生成而不是手写：models.dev 有 300+ provider、每家几十个模型，
手抄一定会写错 base_url。这里只挑出真正会用到的 44 家，并把候选模型
裁到 5 个、按价格分档，输出成 Go 字面量。

维护：
  - 想加一家 → 在下面的 CN/GLOBAL/GATEWAY/LOCAL 列表里加一行
  - 地址写错 → 改那一行的 base_url（不要改生成出来的 presets.go，会被覆盖）
  - 国内几家不在 models.dev 里（百度千帆/腾讯混元/讯飞星火），modelsdev_id 传 None
"""

import json

d = json.load(open('/tmp/modelsdev.json'))

# (our_id, label, kind, base_url, region, modelsdev_id, prefer_prefix, note)
CN = [
 ("deepseek","DeepSeek","openai-compatible","https://api.deepseek.com/v1","cn","deepseek","deepseek",""),
 ("moonshot","Kimi（月之暗面）","openai-compatible","https://api.moonshot.cn/v1","cn","moonshotai-cn","kimi",""),
 ("zhipu","智谱 GLM","openai-compatible","https://open.bigmodel.cn/api/paas/v4","cn","zhipuai","glm",""),
 ("alibaba","通义千问（阿里云百炼）","openai-compatible","https://dashscope.aliyuncs.com/compatible-mode/v1","cn","alibaba-cn","qwen","百炼也代理 deepseek / glm，模型下拉里看到别家属正常"),
 ("volcengine","豆包（火山方舟）","openai-compatible","https://ark.cn-beijing.volces.com/api/v3","cn","volcengine","doubao","模型要填方舟的「接入点 ID」或模型名"),
 ("siliconflow","硅基流动","openai-compatible","https://api.siliconflow.cn/v1","cn","siliconflow-cn","",""),
 ("qiniu","七牛云 AI","openai-compatible","https://api.qnaigc.com/v1","cn","qiniu-ai","",""),
 ("stepfun","阶跃星辰","openai-compatible","https://api.stepfun.com/v1","cn","stepfun","step",""),
 ("minimax","MiniMax","anthropic","https://api.minimax.cn/anthropic/v1","cn","minimax-cn","minimax","用的是 Anthropic 协议端点，不是 OpenAI 兼容"),
 ("modelscope","魔搭 ModelScope","openai-compatible","https://api-inference.modelscope.cn/v1","cn","modelscope","",""),
 ("sensenova","商汤日日新","openai-compatible","https://token.sensenova.cn/v1","cn","sensenova","sense",""),
 ("baidu","百度千帆（文心）","openai-compatible","https://qianfan.baidubce.com/v2","cn",None,"ernie","地址按千帆 v2 兼容模式，模型可点「读取模型」拉取"),
 ("hunyuan","腾讯混元","openai-compatible","https://api.hunyuan.cloud.tencent.com/v1","cn",None,"hunyuan","模型名可点「读取模型」拉取（hunyuan-turbos-latest 等）"),
 ("spark","讯飞星火","openai-compatible","https://spark-api-open.xf-yun.com/v1","cn",None,"generalv","模型名可点「读取模型」拉取（generalv3.5 / 4.0Ultra 等）"),
]

GLOBAL = [
 ("openai","OpenAI","openai-compatible","https://api.openai.com/v1","global","openai","gpt",""),
 ("anthropic","Anthropic Claude","anthropic","https://api.anthropic.com","global","anthropic","claude",""),
 ("gemini","Google Gemini","gemini","","global","google","gemini","Vertex AI 的地址不用改，填 Project + Location、Key 可留空"),
 ("vertex","Google Vertex AI","gemini","","global","google-vertex","gemini","必须填 Project 与 Location（如 us-central1）"),
 ("xai","xAI Grok","openai-compatible","https://api.x.ai/v1","global","xai","grok",""),
 ("groq","Groq","openai-compatible","https://api.groq.com/openai/v1","global","groq","",""),
 ("mistral","Mistral","openai-compatible","https://api.mistral.ai/v1","global","mistral","mistral",""),
 ("cohere","Cohere","openai-compatible","https://api.cohere.ai/compatibility/v1","global","cohere","command",""),
 ("together","Together AI","openai-compatible","https://api.together.xyz/v1","global","togetherai","",""),
 ("deepinfra","DeepInfra","openai-compatible","https://api.deepinfra.com/v1/openai","global","deepinfra","",""),
 ("fireworks","Fireworks AI","openai-compatible","https://api.fireworks.ai/inference/v1","global","fireworks-ai","",""),
 ("perplexity","Perplexity","openai-compatible","https://api.perplexity.ai","global","perplexity","sonar","只做联网问答，不适合普通补全"),
 ("cerebras","Cerebras","openai-compatible","https://api.cerebras.ai/v1","global","cerebras","",""),
 ("nvidia","NVIDIA NIM","openai-compatible","https://integrate.api.nvidia.com/v1","global","nvidia","",""),
 ("huggingface","Hugging Face Router","openai-compatible","https://router.huggingface.co/v1","global","huggingface","",""),
 ("github-copilot","GitHub Copilot","openai-compatible","https://api.githubcopilot.com","global","github-copilot","","需要 Copilot 订阅的 OAuth token，不能直接用 API Key"),
]

GATEWAY = [
 ("openrouter","OpenRouter","openai-compatible","https://openrouter.ai/api/v1","gateway","openrouter","","一个 Key 用几百个模型，适合先拿它试通"),
 ("opencode","OpenCode Zen","openai-compatible","https://opencode.ai/inference/openai/v1","gateway","opencode","",""),
 ("commandcode","CommandCode","openai-compatible","https://api.commandcode.ai/provider/v1","gateway","commandcode","",""),
 ("vercel","Vercel AI Gateway","openai-compatible","https://ai-gateway.vercel.sh/v1","gateway","vercel","",""),
 ("poe","Poe","openai-compatible","https://api.poe.com/v1","gateway","poe","",""),
 ("302ai","302.AI","openai-compatible","https://api.302.ai/v1","gateway","302ai","",""),
 ("aihubmix","AIHubMix","openai-compatible","https://aihubmix.com/v1","gateway","aihubmix","",""),
 ("jiekou","Jiekou.AI","openai-compatible","https://api.jiekou.ai/openai","gateway","jiekou","",""),
 ("zenmux","ZenMux","openai-compatible","https://zenmux.ai/api/v1","gateway","zenmux","",""),
 ("oneapi","自建 one-api / new-api","openai-compatible","http://127.0.0.1:3000/v1","gateway",None,"","把地址改成你自己的中转服务"),
]

LOCAL = [
 ("ollama","本机 Ollama","ollama","http://127.0.0.1:11434","local",None,"","不用 Key，模型列表点一下就能拉"),
 ("lmstudio","LM Studio","openai-compatible","http://127.0.0.1:1234/v1","local","lmstudio","",""),
 ("llamacpp","llama.cpp server","openai-compatible","http://127.0.0.1:8080/v1","local",None,"",""),
 ("vllm","vLLM","openai-compatible","http://127.0.0.1:8000/v1","local",None,"",""),
]

BAD = ("exp", "experimental", "preview", "-alpha", "-beta", ":free",
       # 非对话模型：语音转写 / 合成、向量、重排、实时双工
       "-stt", "stt-", "-tts", "tts-", "whisper", "-voice", "voice-",
       "embed", "rerank", "realtime", "moderation")

def unstable(mid, name):
    blob = (mid + " " + str(name)).lower()
    return any(b in blob for b in BAD)

def tier(cost):
    if not cost: return "balanced"
    out = cost.get("output")
    try: out = float(out)
    except: return "balanced"
    if out <= 0.6: return "fast"
    if out <= 4: return "balanced"
    return "quality"

FAMILIES = ("gpt", "claude", "gemini", "qwen", "deepseek", "glm", "kimi", "llama", "mistral", "grok", "doubao")

def is_chat(m):
    mods = m.get("modalities") or {}
    out = mods.get("output")
    if out and "text" not in out:
        return False          # 排除 TTS / 生图 / 向量模型
    inp = mods.get("input")
    if inp and "text" not in inp:
        # 语音转写（输入只收 audio、输出是 text）会被上面那条放过，
        # 它不能当对话模型用，所以这里要求输入也支持文本。
        return False
    return True

def pick(pid, prefer, n=5):
    if not pid or pid not in d: return []
    models = d[pid].get("models") or {}
    rows = []
    for mid, m in models.items():
        if not is_chat(m): continue
        rows.append((m.get("release_date") or "", mid, m))
    rows.sort(reverse=True)

    picked, seen = [], set()
    def take(pred, limit):
        for _, mid, m in rows:
            if len(picked) >= limit: break
            name = m.get("name") or mid
            if mid in seen or unstable(mid, name): continue
            if pred and not pred(mid.lower()): continue
            seen.add(mid); picked.append((mid, name, tier(m.get("cost"))))
    # 顺序：本家 → 已知家族（网关场景） → 其余
    if prefer: take(lambda x: x.startswith(prefer), n)
    if len(picked) < n: take(lambda x: any(f in x for f in FAMILIES), n)
    if len(picked) < n: take(None, n)
    return picked

def choose_default(models, prefer):
    def pref(mid):
        return (not prefer) or mid.lower().startswith(prefer)
    for want in ("balanced", "fast", "quality"):
        for mid, name, t in models:
            if t == want and pref(mid): return mid
    for mid, name, t in models:
        if t == "balanced": return mid
    return models[0][0] if models else ""

def gostr(s):
    return '"' + s.replace('\\','\\\\').replace('"','\\"') + '"'

lines = []
for group, region in ((CN,"cn"),(GLOBAL,"global"),(GATEWAY,"gateway"),(LOCAL,"local")):
    for our_id, label, kind, base, reg, md, prefer, note in group:
        models = pick(md, prefer)
        doc = (d.get(md, {}) or {}).get("doc") or "" if md else ""
        default = choose_default(models, prefer)

        lines.append("\t{")
        lines.append(f"\t\tID:      {gostr(our_id)},")
        lines.append(f"\t\tLabel:   {gostr(label)},")
        lines.append(f"\t\tKind:    {gostr(kind)},")
        if base: lines.append(f"\t\tBaseURL: {gostr(base)},")
        lines.append(f"\t\tRegion:  {gostr(region)},")
        if doc: lines.append(f"\t\tDoc:     {gostr(doc)},")
        if note: lines.append(f"\t\tNote:    {gostr(note)},")
        if default: lines.append(f"\t\tDefaultModel: {gostr(default)},")
        if models:
            lines.append("\t\tModels: []ModelSpec{")
            for mid, name, t in models:
                lines.append(f"\t\t\t{{ID: {gostr(mid)}, Label: {gostr(name)}, Tier: {gostr(t)}}},")
            lines.append("\t\t},")
        lines.append("\t},")

header = '''package llm

// 内置提供商目录：来自 models.dev 的公开清单（opencode / Crush 等工具用的同一份数据），
// 国内几家按各自官网文档补上兼容端点。
//
// 为什么要这张表：几乎所有厂商都提供 OpenAI 兼容端点，接一家实际上只是
// "填对 base_url + 模型名"。让用户从零背这些地址是纯粹的浪费——选一家，
// 地址和候选模型自动填好，他只需要粘一个 Key。
//
// 这个文件由 presets_gen.py 生成，不要直接改；要改请改脚本再重新生成。
//
// 维护约定：
//   - BaseURL 只到版本段（/v1、/v2、/v3、/compatible-mode/v1），不带具体资源路径
//   - Models 只是候选，真实可用列表以「读取模型」问上游为准
//   - 地址会随厂商变动，限制条件写进 Note，别让用户以为是我们的 bug
type ProviderPreset struct {
	ID      string       `json:"id"`
	Label   string       `json:"label"`
	Kind    ProviderKind `json:"kind"`
	BaseURL string       `json:"base_url,omitempty"`
	// Region cn | global | gateway | local，前端按此分组
	Region string `json:"region"`
	// Doc 官方文档 / 控制台地址，方便查 Key 在哪申请
	Doc  string `json:"doc,omitempty"`
	// Note 一句话提示：特殊鉴权、模型 ID 规则、需要额外字段等
	Note string `json:"note,omitempty"`
	// DefaultModel 选中模板时预填的模型（本家 + 均衡档 + 非实验版）
	DefaultModel string      `json:"default_model,omitempty"`
	Models       []ModelSpec `json:"models,omitempty"`
}

// ProviderPresets 返回全部内置预设（副本，调用方可安全修改）。
func ProviderPresets() []ProviderPreset {
	src := providerPresets
	out := make([]ProviderPreset, len(src))
	copy(out, src)
	return out
}

var providerPresets = []ProviderPreset{
'''

import os
out = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'presets.go')
open(out,'w').write(header + "\n".join(lines) + "\n}\n")
print("written:", len(CN)+len(GLOBAL)+len(GATEWAY)+len(LOCAL))
