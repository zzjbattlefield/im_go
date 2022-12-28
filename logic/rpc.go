package logic

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/smallnest/rpcx/server"
	"github.com/zzjbattlefield/IM_GO/config"
	"github.com/zzjbattlefield/IM_GO/model"
	"github.com/zzjbattlefield/IM_GO/proto"
	"github.com/zzjbattlefield/IM_GO/tools"
)

type LogicRpc struct{}

func (logic *Logic) InitRpcServer() (err error) {
	s := server.NewServer()
	if err = s.RegisterName("LogicRpc", new(LogicRpc), ""); err != nil {
		return err
	}
	err = s.Serve("tcp", "127.0.0.1:6900")
	return err
}

func (rpc *LogicRpc) Register(ctx context.Context, request *proto.RegisterRequest, reply *proto.RegisterResponse) (err error) {
	reply.Code = config.FailReplyCode
	model := new(model.UserModel)
	data := model.CheckHaveUserName(request.UserName)
	if data.ID != 0 {
		return errors.New("用户已经存在 请登录")
	}
	model.Password = tools.Md5(request.Password)
	model.UserName = request.UserName
	userID, err := model.Add()
	if err != nil {
		log.Println(err.Error())
		return err
	}
	if userID == 0 {
		return errors.New("新增用户失败")
	}
	//构建token
	sessionID, err := CreateAuthToken(ctx, model)
	if err != nil {
		return err
	}
	reply.Code = config.SuccessReplyCode
	reply.AuthToken = sessionID
	return
}

func (rpc *LogicRpc) Login(ctx context.Context, request *proto.LoginRequest, reply *proto.LoginResponse) (err error) {
	reply.Code = tools.CodeFail
	userName := request.UserName
	password := tools.Md5(request.Password)
	model := new(model.UserModel)
	userInfo := model.CheckHaveUserName(userName)
	if userInfo.ID == 0 || userInfo.Password != password {
		return errors.New("用户名或密码错误")
	}
	sessionID, err := CreateAuthToken(ctx, userInfo)
	if err != nil {
		return err
	}
	reply.Code = tools.CodeSuccess
	reply.AuthToken = sessionID
	return
}

func (rpc *LogicRpc) Connect(ctx context.Context, request *proto.ConnectRequest, reply *proto.ConnectReply) (err error) {
	logic := new(Logic)
	config.Zap.Infoln("get args authToken is:", request.AuthToken)
	sessionID := tools.GetSessionName(request.AuthToken)
	userInfo, err := RedisClient.HGetAll(ctx, sessionID).Result()
	if err != nil {
		config.Zap.Errorf("redis HGetAll Key: %s error: %s", sessionID, err.Error())
		return err
	}
	reply.UserID, _ = strconv.Atoi(userInfo["userID"])
	if len(userInfo) < 0 {
		reply.UserID = 0
		return
	}
	roomUserkey := logic.GetRoomUserKey(strconv.Itoa(request.RoomID))
	userKey := logic.GetUserKey(userInfo["userID"])

	//绑定当前用户所在的serviceID
	err = RedisClient.Set(ctx, userKey, request.ServiceID, config.RedisBaseValidTime*time.Second).Err()
	if err != nil {
		config.Zap.Errorf("redis Set error: %s", err.Error())
		return
	}
	if RedisClient.HGet(ctx, roomUserkey, userInfo["userID"]).Val() == "" {
		RedisClient.HSet(ctx, roomUserkey, userInfo["userID"], userInfo["Name"])
		RedisClient.Incr(ctx, logic.GetRoomOnlineKey(strconv.Itoa(request.RoomID)))
	}
	config.Zap.Infoln("logic rpc userID", reply.UserID)
	return
}

func (rpc *LogicRpc) DisConnect(ctx context.Context, request *proto.DisConnectRequest, reply *proto.DisConnectReply) (err error) {
	logic := new(Logic)
	roomUserKey := logic.GetRoomUserKey(strconv.Itoa(request.RoomID))
	count, _ := RedisClient.Get(ctx, logic.GetRoomOnlineKey(strconv.Itoa(request.RoomID))).Int()
	if count > 0 {
		RedisClient.Decr(ctx, logic.GetRoomOnlineKey(strconv.Itoa(request.RoomID))).Result()
	}
	if request.UserID > 0 {
		if err = RedisClient.Del(ctx, roomUserKey, strconv.Itoa(request.UserID)).Err(); err != nil {
			config.Zap.Warnf("RedisCli HGetAll roomUserInfo key:%s, err: %s", roomUserKey, err)
		}
		//TODO:广播一下当前的房间信息
	}
	return

}

func (rpc *LogicRpc) CheckAuth(ctx context.Context, request *proto.CheckAuthRequest, reply *proto.CheckAuthReponse) (err error) {
	var tokenVal = make(map[string]string)
	reply.Code = tools.CodeFail
	authToken := request.AuthToken
	tokenVal, err = RedisClient.HGetAll(ctx, authToken).Result()
	if err != nil || len(tokenVal) == 0 {
		config.Zap.Errorw("检测authToken失败", "authToken", authToken)
		return
	}
	reply.UserID, _ = strconv.Atoi(tokenVal["userID"])
	reply.UserName = tokenVal["userName"]
	reply.Code = tools.CodeSuccess
	return
}

func CreateAuthToken(ctx context.Context, user *model.UserModel) (randStr string, err error) {
	randStr = tools.GetRandString(32)
	sessionID := tools.CreateSessionId(randStr)
	sessionData := make(map[string]interface{})
	sessionData["userName"] = user.UserName
	sessionData["userID"] = user.ID
	err = RedisClient.HMSet(ctx, sessionID, sessionData).Err()
	if err != nil {
		return
	}
	err = RedisClient.Expire(ctx, sessionID, 86400*time.Second).Err()
	if err != nil {
		return
	}
	return
}
