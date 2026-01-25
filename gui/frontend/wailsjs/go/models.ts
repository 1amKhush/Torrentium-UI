export namespace main {
	
	export class ConfigData {
	    downloadDir: string;
	    maxUploadRate: number;
	    maxUploadRateHuman: string;
	    maxDownloadRate: number;
	    maxDownloadRateHuman: string;
	    maxParallelDownloads: number;
	    adaptiveParallelDownloads: boolean;
	    enableEndgameMode: boolean;
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
	        this.maxDownloadRate = source["maxDownloadRate"];
	        this.maxDownloadRateHuman = source["maxDownloadRateHuman"];
	        this.maxParallelDownloads = source["maxParallelDownloads"];
	        this.adaptiveParallelDownloads = source["adaptiveParallelDownloads"];
	        this.enableEndgameMode = source["enableEndgameMode"];
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
	export class FilePreviewInfo {
	    cid: string;
	    filename: string;
	    fileType: string;
	    fileSize: number;
	    sizeHuman: string;
	    isPreviewable: boolean;
	    previewUrl?: string;
	    mimeType?: string;
	
	    static createFrom(source: any = {}) {
	        return new FilePreviewInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cid = source["cid"];
	        this.filename = source["filename"];
	        this.fileType = source["fileType"];
	        this.fileSize = source["fileSize"];
	        this.sizeHuman = source["sizeHuman"];
	        this.isPreviewable = source["isPreviewable"];
	        this.previewUrl = source["previewUrl"];
	        this.mimeType = source["mimeType"];
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
	export class PublishResponse {
	    success: boolean;
	    message: string;
	    magnetLink?: string;
	    shareUrl?: string;
	    fileId?: string;
	
	    static createFrom(source: any = {}) {
	        return new PublishResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.magnetLink = source["magnetLink"];
	        this.shareUrl = source["shareUrl"];
	        this.fileId = source["fileId"];
	    }
	}
	export class QueuedDownloadInfo {
	    cid: string;
	    status: string;
	    priority: number;
	    progress: number;
	    bytesDownloaded: number;
	    totalBytes: number;
	    piecesCompleted: number;
	    totalPieces: number;
	    speed: number;
	    speedHuman: string;
	    eta: string;
	    maxBandwidth: number;
	    error: string;
	    addedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new QueuedDownloadInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cid = source["cid"];
	        this.status = source["status"];
	        this.priority = source["priority"];
	        this.progress = source["progress"];
	        this.bytesDownloaded = source["bytesDownloaded"];
	        this.totalBytes = source["totalBytes"];
	        this.piecesCompleted = source["piecesCompleted"];
	        this.totalPieces = source["totalPieces"];
	        this.speed = source["speed"];
	        this.speedHuman = source["speedHuman"];
	        this.eta = source["eta"];
	        this.maxBandwidth = source["maxBandwidth"];
	        this.error = source["error"];
	        this.addedAt = source["addedAt"];
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
	export class WebShareConfigData {
	    portalUrl: string;
	    apiKey: string;
	    defaultVisibility: string;
	    defaultExpiration: number;
	
	    static createFrom(source: any = {}) {
	        return new WebShareConfigData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.portalUrl = source["portalUrl"];
	        this.apiKey = source["apiKey"];
	        this.defaultVisibility = source["defaultVisibility"];
	        this.defaultExpiration = source["defaultExpiration"];
	    }
	}

}

