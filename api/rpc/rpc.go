package rpc

import (
	"context"
	"sync"
	"time"

	"github.com/rpcxio/libkv/store"
	etcdV3 "github.com/rpcxio/rpcx-etcd/client"
	"github.com/smallnest/rpcx/client"
	"github.com/zzjbattlefield/IM_GO/config"
	"github.com/zzjbattlefield/IM_GO/proto"
)

var LogicRpcClient client.XClient
var once sync.Once

type RpcLogic struct{}

var RpcLoginObj *RpcLogic

// 初始化对logicRpc的客户端初始化
func InitLogicRpcClient() {
	once.Do(func() {
		etcdConfig := config.Conf.Common.CommonEtcd
		etcdConfigOption := &store.Config{
			ClientTLS:         nil,
			TLS:               nil,
			ConnectionTimeout: time.Duration(etcdConfig.ConnectionTimeout) * time.Second,
			Bucket:            "",
			PersistConnection: true,
			Username:          etcdConfig.UserName,
			Password:          etcdConfig.Password,
		}
		d, err := etcdV3.NewEtcdV3Discovery(
			etcdConfig.BasePath,
			etcdConfig.ServerPathLogic,
			[]string{etcdConfig.Host},
			true,
			etcdConfigOption,
		)
		if err != nil {
			panic(err)
		}
		LogicRpcClient = client.NewXClient(etcdConfig.ServerPathLogic, client.Failtry, client.RandomSelect, d, client.DefaultOption)

		RpcLoginObj = new(RpcLogic)
	})
	if LogicRpcClient == nil {
		panic("rpc client启动失败")
	}
}

func (rpc *RpcLogic) Login(request *proto.LoginRequest) (code int, authToken string, msg string) {
	reply := new(proto.LoginResponse)
	err := LogicRpcClient.Call(context.Background(), "Login", request, reply)
	if err != nil {
		msg = err.Error()
	}
	authToken = reply.AuthToken
	code = reply.Code
	return
}

func (rpc *RpcLogic) Register(request *proto.RegisterRequest) (code int, authToken string, msg string) {
	reply := new(proto.RegisterResponse)
	err := LogicRpcClient.Call(context.Background(), "Register", request, reply)
	if err != nil {
		msg = err.Error()
	}
	authToken = reply.AuthToken
	code = reply.Code
	return
}

func (rpc *RpcLogic) CheckAuth(request *proto.CheckAuthRequest) (code int, userName string, userID int) {
	reply := new(proto.CheckAuthReponse)
	LogicRpcClient.Call(context.Background(), "CheckAuth", request, reply)
	code = reply.Code
	userName = reply.UserName
	userID = reply.UserID
	config.Zap.Infoln("userID:", userID, " userName:", userName)
	return
}

func (rpc *RpcLogic) GetRoomInfo(request *proto.Send) (code int, msg string) {
	var reply = &proto.SuccessReply{}
	LogicRpcClient.Call(context.Background(), "GetRoomInfo", request, reply)
	code = reply.Code
	msg = reply.Msg
	return
}

func (rpc *RpcLogic) PushRoom(request *proto.Send) (code int, msg string) {
	reply := &proto.SuccessReply{}
	LogicRpcClient.Call(context.Background(), "PushRoom", request, reply)
	code = reply.Code
	msg = reply.Msg
	return
}

func (rpc *RpcLogic) GetUserInfoByUserId(request *proto.GetUserInfoRequest) (code int, userName string) {
	reply := &proto.GetUserInfoResponse{}
	LogicRpcClient.Call(context.Background(), "GetUserInfoByUserId", request, reply)
	return reply.Code, reply.UserName
}

func (rpc *RpcLogic) Push(request *proto.Send) (code int, message string) {
	reply := &proto.SuccessReply{}
	LogicRpcClient.Call(context.Background(), "Push", request, reply)
	return reply.Code, reply.Msg
}
