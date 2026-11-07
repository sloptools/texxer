export namespace main {
	
	export class ModelInfo {
	    path: string;
	    parts: string[];
	    extent: number[];
	    min: number[];
	    diag: number;
	    overlap: number;
	    offsets: number[][];
	    hash: string;
	    details: boolean;
	    texture: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.parts = source["parts"];
	        this.extent = source["extent"];
	        this.min = source["min"];
	        this.diag = source["diag"];
	        this.overlap = source["overlap"];
	        this.offsets = source["offsets"];
	        this.hash = source["hash"];
	        this.details = source["details"];
	        this.texture = source["texture"];
	    }
	}
	export class Photo {
	    id: string;
	    file: string;
	    cuts: string;
	
	    static createFrom(source: any = {}) {
	        return new Photo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.file = source["file"];
	        this.cuts = source["cuts"];
	    }
	}
	export class ProjectFile {
	    path: string;
	    json: string;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.json = source["json"];
	        this.model = source["model"];
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

