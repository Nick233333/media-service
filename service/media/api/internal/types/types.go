// Code generated by goctl. DO NOT EDIT.
package types

type TransformReq struct {
	MediaUrl  string `json:"media_url"`
	Standard  string `json:"standard"`
	NotifyUrl string `json:"notify_url"`
}

type TransformReply struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}
