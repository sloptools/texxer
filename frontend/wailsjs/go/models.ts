export namespace main {
	
	export class ModelInfo {
	    path: string;
	    name: string;
	    parts: string[];
	    extent: number[];
	    min: number[];
	    diag: number;
	    project: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.parts = source["parts"];
	        this.extent = source["extent"];
	        this.min = source["min"];
	        this.diag = source["diag"];
	        this.project = source["project"];
	    }
	}

}

