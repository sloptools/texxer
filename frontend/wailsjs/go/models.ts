export namespace main {
	
	export class ModelInfo {
	    path: string;
	    name: string;
	    parts: string[];
	    extent: number[];
	    min: number[];
	    diag: number;
	    project: string;
	    overlap: number;
	
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
	        this.overlap = source["overlap"];
	    }
	}
	export class UnwrapStats {
	    charts: number;
	    fill: number;
	    density: number;
	
	    static createFrom(source: any = {}) {
	        return new UnwrapStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.charts = source["charts"];
	        this.fill = source["fill"];
	        this.density = source["density"];
	    }
	}
	export class VariantTextures {
	    name: string;
	    color: string;
	    emissive: string;
	
	    static createFrom(source: any = {}) {
	        return new VariantTextures(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.color = source["color"];
	        this.emissive = source["emissive"];
	    }
	}

}

