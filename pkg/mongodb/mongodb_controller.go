package mongodb

import (
	"github.com/globalsign/mgo"
	log "github.com/sirupsen/logrus"
	"message-center/pkg/configuration"
)

type MongoDBController struct {
	connectionStr string
	rootSession   *mgo.Session
	strongSession *mgo.Session
}

func (mc *MongoDBController) Initialize(dcl configuration.ConfigurationLoader) {
	mc.InitializeByConfig(dcl.GetField("DevOps.Mgmt.API", "MongoDB_ConnectionStr"), mgo.Monotonic)
}

func (mc *MongoDBController) InitializeByConfig(connectionStr string, mode mgo.Mode) {
	mc.connectionStr = connectionStr
	s, err := mgo.Dial(mc.connectionStr)
	if err != nil {
		log.Fatalf("Failed to initializing MongoDB instance, error: %s", err.Error())
	}
	mc.rootSession = s
	mc.rootSession.SetMode(mode, true)
	mc.strongSession = s
	mc.strongSession.SetMode(mgo.Strong, true)
}

func (mc *MongoDBController) NewSession() *mgo.Session {
	return mc.rootSession.Copy()
}

// 解决读取都是从主库中进行的问题
func (mc *MongoDBController) NewStrongSession() *mgo.Session {
	return mc.strongSession.Copy()
}

func (mc *MongoDBController) Close() {
	mc.rootSession.Close()
	mc.NewStrongSession()
}
