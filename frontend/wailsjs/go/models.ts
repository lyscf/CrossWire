export namespace app {
	
	export class BanMemberRequest {
	    member_id: string;
	    reason?: string;
	    duration?: number;
	
	    static createFrom(source: any = {}) {
	        return new BanMemberRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.member_id = source["member_id"];
	        this.reason = source["reason"];
	        this.duration = source["duration"];
	    }
	}
	export class ClientConfig {
	    password: string;
	    transport_mode: string;
	    network_interface: string;
	    server_address: string;
	    port: number;
	    nickname: string;
	    avatar: string;
	    auto_reconnect: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ClientConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.password = source["password"];
	        this.transport_mode = source["transport_mode"];
	        this.network_interface = source["network_interface"];
	        this.server_address = source["server_address"];
	        this.port = source["port"];
	        this.nickname = source["nickname"];
	        this.avatar = source["avatar"];
	        this.auto_reconnect = source["auto_reconnect"];
	    }
	}
	export class CreateChallengeRequest {
	    title: string;
	    description: string;
	    category: string;
	    difficulty: string;
	    points: number;
	    flag: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateChallengeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.difficulty = source["difficulty"];
	        this.points = source["points"];
	        this.flag = source["flag"];
	    }
	}
	export class DownloadFileRequest {
	    file_id: string;
	    save_path: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadFileRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_id = source["file_id"];
	        this.save_path = source["save_path"];
	    }
	}
	export class ErrorInfo {
	    code: string;
	    message: string;
	    details?: string;
	
	    static createFrom(source: any = {}) {
	        return new ErrorInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	        this.details = source["details"];
	    }
	}
	export class ExportOptions {
	    include_messages: boolean;
	    include_files: boolean;
	    include_challenges: boolean;
	    include_members: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExportOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.include_messages = source["include_messages"];
	        this.include_files = source["include_files"];
	        this.include_challenges = source["include_challenges"];
	        this.include_members = source["include_members"];
	    }
	}
	export class KickMemberRequest {
	    member_id: string;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new KickMemberRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.member_id = source["member_id"];
	        this.reason = source["reason"];
	    }
	}
	export class NotificationSettings {
	    enabled: boolean;
	    sound: boolean;
	    desktop: boolean;
	    mention_only: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NotificationSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.sound = source["sound"];
	        this.desktop = source["desktop"];
	        this.mention_only = source["mention_only"];
	    }
	}
	export class PinMessageRequest {
	    message_id: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new PinMessageRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message_id = source["message_id"];
	        this.reason = source["reason"];
	    }
	}
	export class Response {
	    success: boolean;
	    data?: any;
	    error?: ErrorInfo;
	
	    static createFrom(source: any = {}) {
	        return new Response(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = source["data"];
	        this.error = this.convertValues(source["error"], ErrorInfo);
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
	export class SearchMessagesRequest {
	    query: string;
	    type?: string;
	    sender_id?: string;
	    start_time?: number;
	    end_time?: number;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchMessagesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.type = source["type"];
	        this.sender_id = source["sender_id"];
	        this.start_time = source["start_time"];
	        this.end_time = source["end_time"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}
	export class SendCodeRequest {
	    code: string;
	    language: string;
	    filename?: string;
	
	    static createFrom(source: any = {}) {
	        return new SendCodeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.language = source["language"];
	        this.filename = source["filename"];
	    }
	}
	export class SendMessageRequest {
	    content: string;
	    type: string;
	    channel_id?: string;
	    reply_to_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new SendMessageRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.type = source["type"];
	        this.channel_id = source["channel_id"];
	        this.reply_to_id = source["reply_to_id"];
	    }
	}
	export class ServerConfig {
	    channel_name: string;
	    password: string;
	    transport_mode: string;
	    network_interface: string;
	    listen_address: string;
	    port: number;
	    max_members: number;
	    max_file_size: number;
	    enable_challenge: boolean;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.channel_name = source["channel_name"];
	        this.password = source["password"];
	        this.transport_mode = source["transport_mode"];
	        this.network_interface = source["network_interface"];
	        this.listen_address = source["listen_address"];
	        this.port = source["port"];
	        this.max_members = source["max_members"];
	        this.max_file_size = source["max_file_size"];
	        this.enable_challenge = source["enable_challenge"];
	        this.description = source["description"];
	    }
	}
	export class SkillDetail {
	    category: string;
	    level: number;
	    experience: number;
	
	    static createFrom(source: any = {}) {
	        return new SkillDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.level = source["level"];
	        this.experience = source["experience"];
	    }
	}
	export class SubmitFlagRequest {
	    challenge_id: string;
	    flag: string;
	
	    static createFrom(source: any = {}) {
	        return new SubmitFlagRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.challenge_id = source["challenge_id"];
	        this.flag = source["flag"];
	    }
	}
	export class UpdateChallengeRequest {
	    title?: string;
	    description?: string;
	    category?: string;
	    difficulty?: string;
	    points?: number;
	    flag?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateChallengeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.difficulty = source["difficulty"];
	        this.points = source["points"];
	        this.flag = source["flag"];
	    }
	}
	export class UpdateProgressRequest {
	    challenge_id: string;
	    progress: number;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateProgressRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.challenge_id = source["challenge_id"];
	        this.progress = source["progress"];
	        this.summary = source["summary"];
	    }
	}
	export class UploadFileRequest {
	    file_path: string;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new UploadFileRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.description = source["description"];
	    }
	}
	export class UserProfile {
	    nickname: string;
	    avatar: string;
	    email: string;
	    bio: string;
	    skills: string[];
	    skill_details?: SkillDetail[];
	    status: string;
	    custom_status: string;
	    theme: string;
	    language: string;
	    notifications: NotificationSettings;
	
	    static createFrom(source: any = {}) {
	        return new UserProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nickname = source["nickname"];
	        this.avatar = source["avatar"];
	        this.email = source["email"];
	        this.bio = source["bio"];
	        this.skills = source["skills"];
	        this.skill_details = this.convertValues(source["skill_details"], SkillDetail);
	        this.status = source["status"];
	        this.custom_status = source["custom_status"];
	        this.theme = source["theme"];
	        this.language = source["language"];
	        this.notifications = this.convertValues(source["notifications"], NotificationSettings);
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

