export namespace main {
	
	export class ConfigData {
	    downloadDir: string;
	    maxUploadRate: number;
	    maxUploadRateHuman: string;
	    logLevel: string;
	    databasePath: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfigData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.downloadDir = source["downloadDir"];
	        this.maxUploadRate = source["maxUploadRate"];
	        this.maxUploadRateHuman = source["maxUploadRateHuman"];
	        this.logLevel = source["logLevel"];
	        this.databasePath = source["databasePath"];
	    }
	}
	export class DownloadInfo {
	    cid: string;
	    filename: string;
	    fileSize: number;
	    sizeHuman: string;
	    downloadPath: string;
	    downloadedAt: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cid = source["cid"];
	        this.filename = source["filename"];
	        this.fileSize = source["fileSize"];
	        this.sizeHuman = source["sizeHuman"];
	        this.downloadPath = source["downloadPath"];
	        this.downloadedAt = source["downloadedAt"];
	        this.status = source["status"];
	    }
	}
	export class LocalFileInfo {
	    cid: string;
	    filename: string;
	    fileSize: number;
	    filePath: string;
	    sizeHuman: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalFileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cid = source["cid"];
	        this.filename = source["filename"];
	        this.fileSize = source["fileSize"];
	        this.filePath = source["filePath"];
	        this.sizeHuman = source["sizeHuman"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class NetworkStatus {
	    peerId: string;
	    listenAddresses: string[];
	    connectedPeers: number;
	    dhtRoutingTable: number;
	    sharedFilesCount: number;
	    isConnected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NetworkStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.peerId = source["peerId"];
	        this.listenAddresses = source["listenAddresses"];
	        this.connectedPeers = source["connectedPeers"];
	        this.dhtRoutingTable = source["dhtRoutingTable"];
	        this.sharedFilesCount = source["sharedFilesCount"];
	        this.isConnected = source["isConnected"];
	    }
	}
	export class PeerInfo {
	    peerId: string;
	    addresses: string[];
	    connected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PeerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.peerId = source["peerId"];
	        this.addresses = source["addresses"];
	        this.connected = source["connected"];
	    }
	}
	export class SearchResult {
	    cid: string;
	    filename: string;
	    providers: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cid = source["cid"];
	        this.filename = source["filename"];
	        this.providers = source["providers"];
	    }
	}
	export class StatsData {
	    totalUploaded: number;
	    totalDownloaded: number;
	    uploadedHuman: string;
	    downloadedHuman: string;
	    chunksServed: number;
	    peersServed: number;
	    filesDownloaded: number;
	    filesShared: number;
	    ratio: number;
	    maxUploadRate: number;
	    maxUploadRateHuman: string;
	
	    static createFrom(source: any = {}) {
	        return new StatsData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalUploaded = source["totalUploaded"];
	        this.totalDownloaded = source["totalDownloaded"];
	        this.uploadedHuman = source["uploadedHuman"];
	        this.downloadedHuman = source["downloadedHuman"];
	        this.chunksServed = source["chunksServed"];
	        this.peersServed = source["peersServed"];
	        this.filesDownloaded = source["filesDownloaded"];
	        this.filesShared = source["filesShared"];
	        this.ratio = source["ratio"];
	        this.maxUploadRate = source["maxUploadRate"];
	        this.maxUploadRateHuman = source["maxUploadRateHuman"];
	    }
	}
	export class UploadProgressInfo {
	    cid: string;
	    bytesUploaded: number;
	    uploadedHuman: string;
	    chunksServed: number;
	    peersServed: number;
	    avgSpeed: string;
	
	    static createFrom(source: any = {}) {
	        return new UploadProgressInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cid = source["cid"];
	        this.bytesUploaded = source["bytesUploaded"];
	        this.uploadedHuman = source["uploadedHuman"];
	        this.chunksServed = source["chunksServed"];
	        this.peersServed = source["peersServed"];
	        this.avgSpeed = source["avgSpeed"];
	    }
	}

}

