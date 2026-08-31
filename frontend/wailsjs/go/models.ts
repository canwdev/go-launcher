export namespace main {
	
	export class AppItem {
	    guid: string;
	    name: string;
	    path: string;
	    icon?: string;
	    runtime_ms?: number;
	    args?: string;
	    working_dir?: string;
	    iconURL?: string;
	    running?: boolean;
	
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
	        this.iconURL = source["iconURL"];
	        this.running = source["running"];
	    }
	}
	export class Settings {
	    auto_minimize: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.auto_minimize = source["auto_minimize"];
	    }
	}
	export class Tab {
	    guid: string;
	    name: string;
	    slots: AppItem[];
	
	    static createFrom(source: any = {}) {
	        return new Tab(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.guid = source["guid"];
	        this.name = source["name"];
	        this.slots = this.convertValues(source["slots"], AppItem);
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
	export class AppStore {
	    version: string;
	    active_tab_guid: string;
	    tabs: Tab[];
	    settings: Settings;
	
	    static createFrom(source: any = {}) {
	        return new AppStore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.active_tab_guid = source["active_tab_guid"];
	        this.tabs = this.convertValues(source["tabs"], Tab);
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
	

}

