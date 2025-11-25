export namespace models {
	
	export class Album {
	    title: string;
	    artist: string;
	    coverUrl: string;
	    url: string;
	    isFree: boolean;
	    isNyp: boolean;
	    price: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new Album(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.coverUrl = source["coverUrl"];
	        this.url = source["url"];
	        this.isFree = source["isFree"];
	        this.isNyp = source["isNyp"];
	        this.price = source["price"];
	        this.status = source["status"];
	    }
	}

}

