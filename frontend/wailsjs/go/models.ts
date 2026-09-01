export namespace main {
	
	export class AppItem {
	    guid: string;
	    name: string;
	    path: string;
	    icon?: string;
	    runtime_ms?: number;
	    args?: string;
	    working_dir?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.guid = source["guid"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.icon = source["icon"];
	        this.runtime_ms = source["runtime_ms"];
	        this.args = source["args"];
	        this.working_dir = source["working_dir"];
	    }
	}
	export class AddResult {
	    items: AppItem[];
	    icons: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new AddResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], AppItem);
	        this.icons = source["icons"];
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
	export class ItemState {
	    running: boolean;
	    runtime_ms: number;
	    icon_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new ItemState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.runtime_ms = source["runtime_ms"];
	        this.icon_url = source["icon_url"];
	    }
	}
	export class Settings {
	    auto_minimize: boolean;
	    absolute_paths: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.auto_minimize = source["auto_minimize"];
	        this.absolute_paths = source["absolute_paths"];
	    }
	}
	export class CategoryNode {
	    guid: string;
	    name: string;
	    slots: string[];
	
	    static createFrom(source: any = {}) {
	        return new CategoryNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.guid = source["guid"];
	        this.name = source["name"];
	        this.slots = source["slots"];
	    }
	}
	export class AppStore {
	    apps: Record<string, AppItem>;
	    categories: CategoryNode[];
	    settings: Settings;
	
	    static createFrom(source: any = {}) {
	        return new AppStore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apps = this.convertValues(source["apps"], AppItem, true);
	        this.categories = this.convertValues(source["categories"], CategoryNode);
	        this.settings = this.convertValues(source["settings"], Settings);
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
	export class AppData {
	    store: AppStore;
	    state: Record<string, ItemState>;
	
	    static createFrom(source: any = {}) {
	        return new AppData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.store = this.convertValues(source["store"], AppStore);
	        this.state = this.convertValues(source["state"], ItemState, true);
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
	
	
	
	export class IconResult {
	    icon: string;
	    icon_url: string;
	
	    static createFrom(source: any = {}) {
	        return new IconResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.icon = source["icon"];
	        this.icon_url = source["icon_url"];
	    }
	}
	

}

