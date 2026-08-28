package main

import (
	"log"
	"os"

	"github.com/mrabhi2k3/telegofer/tl/generator"
	"github.com/mrabhi2k3/telegofer/tl/parser"
)

const coreSchema = `
// Core MTProto 2.0 Service & File Transfer Schema

// Auth Key Creation
resPQ#05162463 nonce:int128 server_nonce:int128 pq:string server_public_key_fingerprints:Vector<long> = ResPQ;
p_q_inner_data_dc#a9f55f95 pq:string p:string q:string nonce:int128 server_nonce:int128 new_nonce:int256 dc:int = P_Q_inner_data;
server_DH_params_ok#d0e8075c nonce:int128 server_nonce:int128 encrypted_answer:string = Server_DH_Params;
server_DH_params_fail#79cb045d nonce:int128 server_nonce:int128 new_nonce_hash:int128 = Server_DH_Params;
server_DH_inner_data#b5890dba nonce:int128 server_nonce:int128 g:int dh_prime:string g_a:string server_time:int = Server_DH_inner_data;
client_DH_inner_data#6643b654 nonce:int128 server_nonce:int128 retry_id:long g_b:string = Client_DH_Inner_Data;
dh_gen_ok#3bcbf734 nonce:int128 server_nonce:int128 new_nonce_hash1:int128 = Set_client_DH_params_answer;
dh_gen_retry#462de825 nonce:int128 server_nonce:int128 new_nonce_hash2:int128 = Set_client_DH_params_answer;
dh_gen_fail#a69dae02 nonce:int128 server_nonce:int128 new_nonce_hash3:int128 = Set_client_DH_params_answer;

// MTProto Service Messages
rpc_error#2144ca19 error_code:int error_message:string = RpcError;
msgs_ack#62d6b459 msg_ids:Vector<long> = MsgsAck;
bad_msg_notification#a7eff811 bad_msg_id:long bad_msg_seqno:int error_code:int = BadMsgNotification;
bad_server_salt#edab447b bad_msg_id:long bad_msg_seqno:int error_code:int new_server_salt:long = BadMsgNotification;
new_session_created#9ec20908 first_msg_id:long unique_id:long server_salt:long = NewSession;
pong#347773c5 msg_id:long ping_id:long = Pong;
destroy_session_ok#e22045fc session_id:long = DestroySessionRes;
destroy_session_none#62d350c9 session_id:long = DestroySessionRes;

// File Upload & Download Core
inputFile#f52ff27f id:long parts:int name:string md5_checksum:string = InputFile;
inputFileBig#fa4f0bb5 id:long parts:int name:string = InputFile;
inputPeerEmpty#7f0783fa = InputPeer;
inputPeerSelf#7da07ec9 = InputPeer;
inputPeerChat#35a95697 chat_id:long = InputPeer;
inputPeerUser#dde8a529 user_id:long access_hash:long = InputPeer;
inputPeerChannel#27bcb61f channel_id:long access_hash:long = InputPeer;

inputFileLocation#dfdaabe1 volume_id:long local_id:int secret:long file_reference:bytes = InputFileLocation;
inputDocumentFileLocation#bad07284 id:long access_hash:long file_reference:bytes thumb_size:string = InputFileLocation;

fileHash#6242c773 offset:long limit:int hash:bytes = FileHash;

storage_fileUnknown#aa963b05 = StorageFileType;
storage_filePartial#40bc6f52 = StorageFileType;
storage_fileJpeg#007efe0e = StorageFileType;
storage_filePng#0a4f63c0 = StorageFileType;
storage_fileMp4#b3cea0e4 = StorageFileType;

upload_file#096a18d3 mtime:int bytes:bytes = UploadFile;
upload_fileCdnRedirect#f18cda44 dc_id:int file_token:bytes encryption_key:bytes encryption_iv:bytes file_hashes:Vector<FileHash> = UploadFile;
upload_cdnFile#a99fca4f bytes:bytes = UploadCdnFile;
upload_cdnFileReuploadNeeded#eea8e486 request_token:bytes = UploadCdnFile;

---functions---

req_pq_multi#be7e8ef1 nonce:int128 = ResPQ;
req_DH_params#d712e4be nonce:int128 server_nonce:int128 p:string q:string public_key_fingerprint:long encrypted_data:string = Server_DH_Params;
set_client_DH_params#f5045f1f nonce:int128 server_nonce:int128 encrypted_data:string = Set_client_DH_params_answer;
ping#7abe77ec ping_id:long = Pong;
destroy_session#e7512126 session_id:long = DestroySessionRes;

upload_saveFilePart#b304a621 file_id:long file_part:int bytes:bytes = Bool;
upload_saveBigFilePart#de7b673d file_id:long file_part:int file_total_parts:int bytes:bytes = Bool;
upload_getFile#be250529 flags:# precise:flags.0?true cdn_supported:flags.1?true location:InputFileLocation offset:long limit:int = UploadFile;
upload_getCdnFile#20005570 file_token:bytes offset:long limit:int = UploadCdnFile;
upload_reuploadCdnFile#9b2754a8 file_token:bytes request_token:bytes = Vector<FileHash>;
upload_getCdnFileHashes#4e545469 file_token:bytes offset:long = Vector<FileHash>;
`

func main() {
	schema, err := parser.Parse(coreSchema)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	gen := generator.New("generated")
	code, err := gen.Generate(schema)
	if err != nil {
		log.Fatalf("Generate error: %v", err)
	}

	target := "tl/generated/core.go"
	if err := os.WriteFile(target, code, 0644); err != nil {
		log.Fatalf("WriteFile error: %v", err)
	}

	log.Printf("Successfully generated %s (%d bytes)", target, len(code))
}
