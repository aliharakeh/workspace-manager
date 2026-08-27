export namespace gitgraph {
	
	export class BranchInfo {
	    name: string;
	    updated?: string;
	
	    static createFrom(source: any = {}) {
	        return new BranchInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.updated = source["updated"];
	    }
	}
	export class CommitNode {
	    hash: string;
	    branch: string;
	    on?: string[];
	    parents?: string[];
	    timestamp: string;
	    author: string;
	    subject: string;
	    isMerge: boolean;
	    tags?: string[];
	    lanes?: string[];
	
	    static createFrom(source: any = {}) {
	        return new CommitNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.branch = source["branch"];
	        this.on = source["on"];
	        this.parents = source["parents"];
	        this.timestamp = source["timestamp"];
	        this.author = source["author"];
	        this.subject = source["subject"];
	        this.isMerge = source["isMerge"];
	        this.tags = source["tags"];
	        this.lanes = source["lanes"];
	    }
	}
	export class MergeEvent {
	    hash: string;
	    kind?: string;
	    sourceBranch: string;
	    targetBranch: string;
	    sourceHash: string;
	    timestamp: string;
	    author: string;
	    subject: string;
	    commitCount: number;
	
	    static createFrom(source: any = {}) {
	        return new MergeEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.kind = source["kind"];
	        this.sourceBranch = source["sourceBranch"];
	        this.targetBranch = source["targetBranch"];
	        this.sourceHash = source["sourceHash"];
	        this.timestamp = source["timestamp"];
	        this.author = source["author"];
	        this.subject = source["subject"];
	        this.commitCount = source["commitCount"];
	    }
	}
	export class RemoteInfo {
	    name: string;
	    url: string;
	    web?: string;
	    host?: string;
	    ssh: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RemoteInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.web = source["web"];
	        this.host = source["host"];
	        this.ssh = source["ssh"];
	    }
	}
	export class RepoGraph {
	    path: string;
	    commitUrl?: string;
	    branches: string[];
	    commits: CommitNode[];
	    merges: MergeEvent[];
	
	    static createFrom(source: any = {}) {
	        return new RepoGraph(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.commitUrl = source["commitUrl"];
	        this.branches = source["branches"];
	        this.commits = this.convertValues(source["commits"], CommitNode);
	        this.merges = this.convertValues(source["merges"], MergeEvent);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace types {
	
	export class AIActivateInput {
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new AIActivateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	    }
	}
	export class AIChatInput {
	    system?: string;
	    prompt: string;
	
	    static createFrom(source: any = {}) {
	        return new AIChatInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.system = source["system"];
	        this.prompt = source["prompt"];
	    }
	}
	export class AIChatResult {
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new AIChatResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	    }
	}
	export class AIConnectionInfo {
	    name: string;
	    provider: string;
	    baseURL?: string;
	    model?: string;
	    hasApiKey: boolean;
	    temperature?: number;
	
	    static createFrom(source: any = {}) {
	        return new AIConnectionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.baseURL = source["baseURL"];
	        this.model = source["model"];
	        this.hasApiKey = source["hasApiKey"];
	        this.temperature = source["temperature"];
	    }
	}
	export class AIConfigInfo {
	    providers: AIConnectionInfo[];
	    active: string;
	
	    static createFrom(source: any = {}) {
	        return new AIConfigInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providers = this.convertValues(source["providers"], AIConnectionInfo);
	        this.active = source["active"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class AIProviderConfigInput {
	    name: string;
	    provider: string;
	    baseURL?: string;
	    apiKey?: string;
	    model?: string;
	    temperature?: number;
	    clearApiKey?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AIProviderConfigInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.baseURL = source["baseURL"];
	        this.apiKey = source["apiKey"];
	        this.model = source["model"];
	        this.temperature = source["temperature"];
	        this.clearApiKey = source["clearApiKey"];
	    }
	}
	export class AITestInput {
	    provider: string;
	    baseURL?: string;
	    apiKey?: string;
	    model?: string;
	    temperature?: number;
	
	    static createFrom(source: any = {}) {
	        return new AITestInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.baseURL = source["baseURL"];
	        this.apiKey = source["apiKey"];
	        this.model = source["model"];
	        this.temperature = source["temperature"];
	    }
	}
	export class AITestResult {
	    ok: boolean;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new AITestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.text = source["text"];
	    }
	}
	export class App {
	    id: number;
	    workspace_id: number;
	    name: string;
	    project_path: string;
	    active_config_set_id?: number;
	    active_config_set_name?: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new App(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.workspace_id = source["workspace_id"];
	        this.name = source["name"];
	        this.project_path = source["project_path"];
	        this.active_config_set_id = source["active_config_set_id"];
	        this.active_config_set_name = source["active_config_set_name"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class AppAIToolCall {
	    name: string;
	    input: any;
	    output: any;
	
	    static createFrom(source: any = {}) {
	        return new AppAIToolCall(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.input = source["input"];
	        this.output = source["output"];
	    }
	}
	export class AppAIChatTurn {
	    role: string;
	    text: string;
	    tools?: AppAIToolCall[];
	
	    static createFrom(source: any = {}) {
	        return new AppAIChatTurn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.text = source["text"];
	        this.tools = this.convertValues(source["tools"], AppAIToolCall);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppAIChatInput {
	    appId: number;
	    configSetId: number;
	    history: AppAIChatTurn[];
	    instruction: string;
	
	    static createFrom(source: any = {}) {
	        return new AppAIChatInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appId = source["appId"];
	        this.configSetId = source["configSetId"];
	        this.history = this.convertValues(source["history"], AppAIChatTurn);
	        this.instruction = source["instruction"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppAIRunCommand {
	    label?: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new AppAIRunCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.command = source["command"];
	    }
	}
	export class AppAIRunPatch {
	    mode?: string;
	    commands?: AppAIRunCommand[];
	
	    static createFrom(source: any = {}) {
	        return new AppAIRunPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.commands = this.convertValues(source["commands"], AppAIRunCommand);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppAITemplatePatch {
	    file_path: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new AppAITemplatePatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.content = source["content"];
	    }
	}
	export class AppAIEnvUpsert {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new AppAIEnvUpsert(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class AppAIEnvPatch {
	    upsert?: AppAIEnvUpsert[];
	    delete?: string[];
	
	    static createFrom(source: any = {}) {
	        return new AppAIEnvPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.upsert = this.convertValues(source["upsert"], AppAIEnvUpsert);
	        this.delete = source["delete"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppAIPatch {
	    message: string;
	    env?: AppAIEnvPatch;
	    templates?: AppAITemplatePatch[];
	    run?: AppAIRunPatch;
	
	    static createFrom(source: any = {}) {
	        return new AppAIPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.env = this.convertValues(source["env"], AppAIEnvPatch);
	        this.templates = this.convertValues(source["templates"], AppAITemplatePatch);
	        this.run = this.convertValues(source["run"], AppAIRunPatch);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppAIChatResult {
	    text: string;
	    patch: AppAIPatch;
	    toolCalls?: AppAIToolCall[];
	
	    static createFrom(source: any = {}) {
	        return new AppAIChatResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.patch = this.convertValues(source["patch"], AppAIPatch);
	        this.toolCalls = this.convertValues(source["toolCalls"], AppAIToolCall);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	
	export class AppCreateInput {
	    name: string;
	    project_path: string;
	
	    static createFrom(source: any = {}) {
	        return new AppCreateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.project_path = source["project_path"];
	    }
	}
	export class AppUpdateInput {
	    name?: string;
	    project_path?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppUpdateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.project_path = source["project_path"];
	    }
	}
	export class ConfigSet {
	    id: number;
	    app_id: number;
	    name: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfigSet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.app_id = source["app_id"];
	        this.name = source["name"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class ConfigSetActivateResult {
	    id: number;
	    app_id: number;
	    name: string;
	    app: App;
	
	    static createFrom(source: any = {}) {
	        return new ConfigSetActivateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.app_id = source["app_id"];
	        this.name = source["name"];
	        this.app = this.convertValues(source["app"], App);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CopyParts {
	    env?: any;
	    templates?: any;
	    run?: any;
	
	    static createFrom(source: any = {}) {
	        return new CopyParts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.env = source["env"];
	        this.templates = source["templates"];
	        this.run = source["run"];
	    }
	}
	export class ConfigSetCreateInput {
	    name: string;
	    copy_from_id?: number;
	    activate?: boolean;
	    parts?: CopyParts;
	
	    static createFrom(source: any = {}) {
	        return new ConfigSetCreateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.copy_from_id = source["copy_from_id"];
	        this.activate = source["activate"];
	        this.parts = this.convertValues(source["parts"], CopyParts);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RunCommand {
	    id: number;
	    run_config_id: number;
	    label?: string;
	    command: string;
	    sort_order: number;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new RunCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.run_config_id = source["run_config_id"];
	        this.label = source["label"];
	        this.command = source["command"];
	        this.sort_order = source["sort_order"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class RunConfig {
	    id: number;
	    config_set_id: number;
	    mode: string;
	    created_at: string;
	    updated_at: string;
	    commands: RunCommand[];
	
	    static createFrom(source: any = {}) {
	        return new RunConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.config_set_id = source["config_set_id"];
	        this.mode = source["mode"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	        this.commands = this.convertValues(source["commands"], RunCommand);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Template {
	    id: number;
	    config_set_id: number;
	    file_path: string;
	    content: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Template(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.config_set_id = source["config_set_id"];
	        this.file_path = source["file_path"];
	        this.content = source["content"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class EnvVar {
	    id: number;
	    config_set_id: number;
	    key: string;
	    value: string;
	    include_in_ai: boolean;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new EnvVar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.config_set_id = source["config_set_id"];
	        this.key = source["key"];
	        this.value = source["value"];
	        this.include_in_ai = source["include_in_ai"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class ConfigSetDetail {
	    id: number;
	    app_id: number;
	    name: string;
	    created_at: string;
	    updated_at: string;
	    env_vars: EnvVar[];
	    templates: Template[];
	    run_config?: RunConfig;
	
	    static createFrom(source: any = {}) {
	        return new ConfigSetDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.app_id = source["app_id"];
	        this.name = source["name"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	        this.env_vars = this.convertValues(source["env_vars"], EnvVar);
	        this.templates = this.convertValues(source["templates"], Template);
	        this.run_config = this.convertValues(source["run_config"], RunConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConfigSetUpdateInput {
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfigSetUpdateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	    }
	}
	
	
	export class EnvVarCreateInput {
	    key: string;
	    value?: string;
	    include_in_ai?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EnvVarCreateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.include_in_ai = source["include_in_ai"];
	    }
	}
	export class EnvVarUpdateInput {
	    key?: string;
	    value?: string;
	    include_in_ai?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EnvVarUpdateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.include_in_ai = source["include_in_ai"];
	    }
	}
	export class FsPickFileInput {
	    startDir?: string;
	    appId?: number;
	
	    static createFrom(source: any = {}) {
	        return new FsPickFileInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.startDir = source["startDir"];
	        this.appId = source["appId"];
	    }
	}
	export class FsPickFolderInput {
	    startDir?: string;
	
	    static createFrom(source: any = {}) {
	        return new FsPickFolderInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.startDir = source["startDir"];
	    }
	}
	export class GitGraphLoadInput {
	    branches: string[];
	    since: string;
	    until: string;
	
	    static createFrom(source: any = {}) {
	        return new GitGraphLoadInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.branches = source["branches"];
	        this.since = source["since"];
	        this.until = source["until"];
	    }
	}
	export class ImportTemplateResult {
	    id: number;
	    file_path: string;
	    created: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ImportTemplateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.file_path = source["file_path"];
	        this.created = source["created"];
	    }
	}
	export class ImportEnvResult {
	    cancelled: boolean;
	    path?: string;
	    format?: string;
	    imported?: number;
	    vars?: EnvVar[];
	    template?: ImportTemplateResult;
	
	    static createFrom(source: any = {}) {
	        return new ImportEnvResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cancelled = source["cancelled"];
	        this.path = source["path"];
	        this.format = source["format"];
	        this.imported = source["imported"];
	        this.vars = this.convertValues(source["vars"], EnvVar);
	        this.template = this.convertValues(source["template"], ImportTemplateResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ListeningProcess {
	    port: number;
	    pid: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new ListeningProcess(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.pid = source["pid"];
	        this.name = source["name"];
	    }
	}
	export class LogEvent {
	    type: string;
	    appId?: number;
	    commandId: number;
	    stream: string;
	    text: string;
	    ts: number;
	
	    static createFrom(source: any = {}) {
	        return new LogEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.appId = source["appId"];
	        this.commandId = source["commandId"];
	        this.stream = source["stream"];
	        this.text = source["text"];
	        this.ts = source["ts"];
	    }
	}
	export class Ok {
	    ok: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Ok(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	    }
	}
	export class PickFileResult {
	    cancelled: boolean;
	    path?: string;
	    relative_path?: string;
	    content?: string;
	
	    static createFrom(source: any = {}) {
	        return new PickFileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cancelled = source["cancelled"];
	        this.path = source["path"];
	        this.relative_path = source["relative_path"];
	        this.content = source["content"];
	    }
	}
	export class PickFolderResult {
	    cancelled: boolean;
	    path?: string;
	
	    static createFrom(source: any = {}) {
	        return new PickFolderResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cancelled = source["cancelled"];
	        this.path = source["path"];
	    }
	}
	export class PortsKillResult {
	    ok: boolean;
	    pid: number;
	
	    static createFrom(source: any = {}) {
	        return new PortsKillResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.pid = source["pid"];
	    }
	}
	export class PortsListResult {
	    min: number;
	    max: number;
	    processes: ListeningProcess[];
	
	    static createFrom(source: any = {}) {
	        return new PortsListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.min = source["min"];
	        this.max = source["max"];
	        this.processes = this.convertValues(source["processes"], ListeningProcess);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProcessState {
	    commandId: number;
	    label: string;
	    command: string;
	    status: string;
	    exitCode?: number;
	    pid?: number;
	    urls: string[];
	
	    static createFrom(source: any = {}) {
	        return new ProcessState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.commandId = source["commandId"];
	        this.label = source["label"];
	        this.command = source["command"];
	        this.status = source["status"];
	        this.exitCode = source["exitCode"];
	        this.pid = source["pid"];
	        this.urls = source["urls"];
	    }
	}
	export class ReadAppFileResult {
	    ok: boolean;
	    content: string;
	    relative_path: string;
	
	    static createFrom(source: any = {}) {
	        return new ReadAppFileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.content = source["content"];
	        this.relative_path = source["relative_path"];
	    }
	}
	export class ReadyUrlPattern {
	    id: number;
	    key?: string;
	    label: string;
	    pattern: string;
	    flags: string;
	    sort_order: number;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new ReadyUrlPattern(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.key = source["key"];
	        this.label = source["label"];
	        this.pattern = source["pattern"];
	        this.flags = source["flags"];
	        this.sort_order = source["sort_order"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class ReadyUrlPatternCreateInput {
	    label: string;
	    pattern: string;
	    flags?: string;
	
	    static createFrom(source: any = {}) {
	        return new ReadyUrlPatternCreateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.pattern = source["pattern"];
	        this.flags = source["flags"];
	    }
	}
	export class ReadyUrlPatternUpdateInput {
	    label?: string;
	    pattern?: string;
	    flags?: string;
	
	    static createFrom(source: any = {}) {
	        return new ReadyUrlPatternUpdateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.pattern = source["pattern"];
	        this.flags = source["flags"];
	    }
	}
	
	export class RunCommandInput {
	    label?: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new RunCommandInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.command = source["command"];
	    }
	}
	
	export class RunConfigSaveInput {
	    mode?: string;
	    commands: RunCommandInput[];
	
	    static createFrom(source: any = {}) {
	        return new RunConfigSaveInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.commands = this.convertValues(source["commands"], RunCommandInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StatusEvent {
	    type: string;
	    sessionId: string;
	    appId: number;
	    running: boolean;
	    processes: ProcessState[];
	    error?: string;
	    ts: number;
	
	    static createFrom(source: any = {}) {
	        return new StatusEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.sessionId = source["sessionId"];
	        this.appId = source["appId"];
	        this.running = source["running"];
	        this.processes = this.convertValues(source["processes"], ProcessState);
	        this.error = source["error"];
	        this.ts = source["ts"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RunnerLogsSnapshot {
	    status: StatusEvent;
	    logs: LogEvent[];
	
	    static createFrom(source: any = {}) {
	        return new RunnerLogsSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = this.convertValues(source["status"], StatusEvent);
	        this.logs = this.convertValues(source["logs"], LogEvent);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class TemplateCreateInput {
	    file_path: string;
	    content?: string;
	
	    static createFrom(source: any = {}) {
	        return new TemplateCreateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.content = source["content"];
	    }
	}
	export class TemplateUpdateInput {
	    file_path?: string;
	    content?: string;
	
	    static createFrom(source: any = {}) {
	        return new TemplateUpdateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.content = source["content"];
	    }
	}
	export class ValidatePathResult {
	    ok: boolean;
	    path?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ValidatePathResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.path = source["path"];
	        this.error = source["error"];
	    }
	}
	export class Workspace {
	    id: number;
	    name: string;
	    icon?: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Workspace(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class WorkspaceCreateInput {
	    name: string;
	    icon?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceCreateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.icon = source["icon"];
	    }
	}
	export class WorkspaceUpdateInput {
	    name?: string;
	    icon?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceUpdateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.icon = source["icon"];
	    }
	}

}

