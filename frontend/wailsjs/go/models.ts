export namespace main {
	
	export class LauncherItemView {
	    title: string;
	    iconURL: string;
	    runtime_ms: number;
	    running: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LauncherItemView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.iconURL = source["iconURL"];
	        this.runtime_ms = source["runtime_ms"];
	        this.running = source["running"];
	    }
	}

}

