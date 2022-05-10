package handler

import (
	"context"

	"github.com/ketches/ketches/internal/utils"
)

type RespCode uint16
type RespErrMsg string

const (
	CodeSuccess  RespCode = 200
	CodeNoData   RespCode = 204
	CodeNotFound RespCode = 404
	CodeError    RespCode = 400
)
const ()

type Resp struct {
	Code  RespCode    `json:"code"`
	Data  interface{} `json:"data"`
	Error error       `json:"error"`
}

type pageData struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total""`
}

func Error(err error) Resp {
	return Resp{
		Code:  CodeError,
		Error: err,
	}
}

func NotFound() Resp {
	return Resp{
		Code:  CodeNotFound,
		Error: errs.NotFoundError,
	}
}

func Ok() Resp {
	return Resp{
		Code: CodeSuccess,
	}
}

func Success(data interface{}) Resp {
	return Resp{
		Code: CodeSuccess,
		Data: data,
	}
}

func NoData() Resp {
	return Resp{
		Code: CodeNoData,
	}
}

func Page(data interface{}, total int64) Resp {
	if data == nil || total == 0 {
		return NoData()
	}

	return Resp{
		Code: CodeSuccess,
		Data: pageData{
			List:  data,
			Total: total,
		},
	}
}

func UserId(ctx context.Context) uint64 {
	return utils.StringToUint64(ctx.Value("UserID").(string))
}

func ClusterId(ctx context.Context) string {
	return ctx.Value("ClusterID").(string)
}
