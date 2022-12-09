package rpc

// not support "earliest" and "pending"
func (s *RpcServer) GetBalance(method string, params ...any) any {

	return nil
}
