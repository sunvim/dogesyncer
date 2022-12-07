package rpc

type RpcFunc func(method string, params ...interface{}) []byte

var (
	methodMap = map[string]RpcFunc{
		"eth_getBlockNumber": GetBlockNumber,
		"eth_getBalance":     GetBalance,
	}
)
