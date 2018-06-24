package model

type ConfigInfo struct {
	JavaHome 		string 		`json:"java_home"`
	ApkTool  		string 		`json:"apk_tool"`
	KeyStore		string 		`json:"key_store"`
	StoreAlias		string		`json:"store_alias"`
	StorePwd		string		`json:"store_password"`
}
