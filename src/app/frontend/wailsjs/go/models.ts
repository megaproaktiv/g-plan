export namespace main {
	
	export class AssignmentDTO {
	    wp: string;
	    percent: number;
	    consumed: string;
	
	    static createFrom(source: any = {}) {
	        return new AssignmentDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wp = source["wp"];
	        this.percent = source["percent"];
	        this.consumed = source["consumed"];
	    }
	}
	export class DepDTO {
	    from: string;
	    to: string;
	    kind: string;
	    fromAnchor: string;
	    toAnchor: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new DepDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.kind = source["kind"];
	        this.fromAnchor = source["fromAnchor"];
	        this.toAnchor = source["toAnchor"];
	        this.color = source["color"];
	    }
	}
	export class DetailDTO {
	    id: string;
	    name: string;
	    desc: string;
	    role: string;
	    start: string;
	    end: string;
	    est: string;
	
	    static createFrom(source: any = {}) {
	        return new DetailDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.desc = source["desc"];
	        this.role = source["role"];
	        this.start = source["start"];
	        this.end = source["end"];
	        this.est = source["est"];
	    }
	}
	export class MilestoneDTO {
	    id: string;
	    name: string;
	    date: string;
	
	    static createFrom(source: any = {}) {
	        return new MilestoneDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.date = source["date"];
	    }
	}
	export class PhaseDTO {
	    id: string;
	    name: string;
	    start: string;
	    end: string;
	
	    static createFrom(source: any = {}) {
	        return new PhaseDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.start = source["start"];
	        this.end = source["end"];
	    }
	}
	export class RoleDTO {
	    name: string;
	    description: string;
	    totalEst: string;
	    assignments: AssignmentDTO[];
	
	    static createFrom(source: any = {}) {
	        return new RoleDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.totalEst = source["totalEst"];
	        this.assignments = this.convertValues(source["assignments"], AssignmentDTO);
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
	export class WorkPackDTO {
	    id: string;
	    num: number;
	    name: string;
	    desc: string;
	    start: string;
	    end: string;
	    est: string;
	    mode: string;
	    summary: boolean;
	    color: string;
	    cluster: string;
	    details: DetailDTO[];
	
	    static createFrom(source: any = {}) {
	        return new WorkPackDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.num = source["num"];
	        this.name = source["name"];
	        this.desc = source["desc"];
	        this.start = source["start"];
	        this.end = source["end"];
	        this.est = source["est"];
	        this.mode = source["mode"];
	        this.summary = source["summary"];
	        this.color = source["color"];
	        this.cluster = source["cluster"];
	        this.details = this.convertValues(source["details"], DetailDTO);
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
	export class ProjectDTO {
	    title: string;
	    description: string;
	    unit: string;
	    hoursPerDay: number;
	    start: string;
	    end: string;
	    totalEst: string;
	    milestones: MilestoneDTO[];
	    phases: PhaseDTO[];
	    packages: WorkPackDTO[];
	    deps: DepDTO[];
	    roles: RoleDTO[];
	    sectionColors: Record<string, string>;
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ProjectDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.description = source["description"];
	        this.unit = source["unit"];
	        this.hoursPerDay = source["hoursPerDay"];
	        this.start = source["start"];
	        this.end = source["end"];
	        this.totalEst = source["totalEst"];
	        this.milestones = this.convertValues(source["milestones"], MilestoneDTO);
	        this.phases = this.convertValues(source["phases"], PhaseDTO);
	        this.packages = this.convertValues(source["packages"], WorkPackDTO);
	        this.deps = this.convertValues(source["deps"], DepDTO);
	        this.roles = this.convertValues(source["roles"], RoleDTO);
	        this.sectionColors = source["sectionColors"];
	        this.warnings = source["warnings"];
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

export namespace plan {
	
	export class Paths {
	    MMD: string;
	    Roles: string;
	    MD: string;
	
	    static createFrom(source: any = {}) {
	        return new Paths(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.MMD = source["MMD"];
	        this.Roles = source["Roles"];
	        this.MD = source["MD"];
	    }
	}

}

