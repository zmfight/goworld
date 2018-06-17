package dispatchercluster

import (
	"github.com/xiaonanln/goworld/engine/common"
	"github.com/xiaonanln/goworld/engine/config"
	"github.com/xiaonanln/goworld/engine/dispatchercluster/dispatcherclient"
	"github.com/xiaonanln/goworld/engine/gwlog"
	"github.com/xiaonanln/goworld/engine/netutil"
	"github.com/xiaonanln/goworld/engine/proto"
)

var (
	dispatcherConns []*dispatcherclient.DispatcherConnMgr
	dispatcherNum   int
	gid             uint16
)

func Initialize(_gid uint16, dctype dispatcherclient.DispatcherClientType, isRestoreGame, isBanBootEntity bool, delegate dispatcherclient.IDispatcherClientDelegate) {
	gid = _gid
	if gid == 0 {
		gwlog.Fatalf("gid is 0")
	}

	dispIds := config.GetDispatcherIDs()
	dispatcherNum = len(dispIds)
	if dispatcherNum == 0 {
		gwlog.Fatalf("dispatcher number is 0")
	}

	dispatcherConns = make([]*dispatcherclient.DispatcherConnMgr, dispatcherNum)
	for _, dispid := range dispIds {
		dispatcherConns[dispid-1] = dispatcherclient.NewDispatcherConnMgr(gid, dctype, dispid, isRestoreGame, isBanBootEntity, delegate)
	}
	for _, dispConn := range dispatcherConns {
		dispConn.Connect()
	}
}

func SendNotifyDestroyEntity(id common.EntityID) error {
	return SelectByEntityID(id).SendNotifyDestroyEntity(id)
}

func SendClearClientFilterProp(gateid uint16, clientid common.ClientID) (err error) {
	return SelectByGateID(gateid).SendClearClientFilterProp(gateid, clientid)
}

func SendSetClientFilterProp(gateid uint16, clientid common.ClientID, key, val string) (err error) {
	return SelectByGateID(gateid).SendSetClientFilterProp(gateid, clientid, key, val)
}

func SendMigrateRequest(entityID common.EntityID, spaceID common.EntityID, spaceGameID uint16) error {
	return SelectByEntityID(entityID).SendMigrateRequest(entityID, spaceID, spaceGameID)
}

func SendRealMigrate(eid common.EntityID, targetGame uint16, data []byte) error {
	return SelectByEntityID(eid).SendRealMigrate(eid, targetGame, data)
}
func SendCallFilterClientProxies(op proto.FilterClientsOpType, key, val string, method string, args []interface{}) (anyerror error) {
	// TODO: broadcast one packet instead of sending multiple packets
	for _, dcm := range dispatcherConns {
		err := dcm.GetDispatcherClientForSend().SendCallFilterClientProxies(op, key, val, method, args)
		if err != nil && anyerror == nil {
			anyerror = err
		}
	}
	return
}

func SendNotifyCreateEntity(id common.EntityID) error {
	if gid != 0 {
		return SelectByEntityID(id).SendNotifyCreateEntity(id)
	} else {
		// goes here when creating nil space or restoring freezed entities
		return nil
	}
}

func SendLoadEntityAnywhere(typeName string, entityID common.EntityID) error {
	return SelectByEntityID(entityID).SendLoadEntitySomewhere(typeName, entityID, 0)
}

func SendLoadEntityOnGame(typeName string, entityID common.EntityID, gameid uint16) error {
	return SelectByEntityID(entityID).SendLoadEntitySomewhere(typeName, entityID, gameid)
}

func SendCreateEntityAnywhere(entityid common.EntityID, typeName string, data map[string]interface{}) error {
	return SelectByEntityID(entityid).SendCreateEntityAnywhere(entityid, typeName, data)
}

func SendStartFreezeGame(gameid uint16) (anyerror error) {
	// TODO: broadcast one packet instead of sending multiple packets
	for _, dcm := range dispatcherConns {
		err := dcm.GetDispatcherClientForSend().SendStartFreezeGame(gameid)
		if err != nil {
			anyerror = err
		}
	}
	return
}

func SendSrvdisRegister(srvid string, info string, force bool) {
	SelectBySrvID(srvid).SendSrvdisRegister(srvid, info, force)
}

func SendCallNilSpaces(exceptGameID uint16, method string, args []interface{}) (anyerror error) {
	// construct one packet for multiple sending
	packet := netutil.NewPacket()
	packet.AppendUint16(proto.MT_CALL_NIL_SPACES)
	packet.AppendUint16(exceptGameID)
	packet.AppendVarStr(method)
	packet.AppendArgs(args)

	for _, dcm := range dispatcherConns {
		err := dcm.GetDispatcherClientForSend().SendPacket(packet)
		if err != nil {
			anyerror = err
		}
	}

	packet.Release()
	return
}

func EntityIDToDispatcherID(entityid common.EntityID) uint16 {
	return uint16((hashEntityID(entityid) % dispatcherNum) + 1)
}

func SrvIDToDispatcherID(srvid string) uint16 {
	return uint16((hashSrvID(srvid) % dispatcherNum) + 1)
}

func SelectByEntityID(entityid common.EntityID) *dispatcherclient.DispatcherClient {
	idx := hashEntityID(entityid) % dispatcherNum
	return dispatcherConns[idx].GetDispatcherClientForSend()
}

func SelectByGateID(gateid uint16) *dispatcherclient.DispatcherClient {
	idx := hashGateID(gateid) % dispatcherNum
	return dispatcherConns[idx].GetDispatcherClientForSend()
}

func SelectByDispatcherID(dispid uint16) *dispatcherclient.DispatcherClient {
	return dispatcherConns[dispid-1].GetDispatcherClientForSend()
}

func SelectBySrvID(srvid string) *dispatcherclient.DispatcherClient {
	idx := hashSrvID(srvid) % dispatcherNum
	return dispatcherConns[idx].GetDispatcherClientForSend()
}

func Select(dispidx int) *dispatcherclient.DispatcherClient {
	return dispatcherConns[dispidx].GetDispatcherClientForSend()
}
