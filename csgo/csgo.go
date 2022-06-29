package csgo

import (
	"GoInspect/csgo/protocol/protobuf"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/Philipp15b/go-steam/v3"
	"github.com/Philipp15b/go-steam/v3/protocol/gamecoordinator"
)

const AppId = 730

func Param(s string) *uint64 {
	i, _ := strconv.Atoi(s)
	addr := uint64(i)
	return &addr
}

func DeParam(param *uint64) string {
	return strconv.FormatInt(int64(*param), 10)
}

type CSGO struct {
	client *steam.Client
}

func New(client *steam.Client) *CSGO {
	c := &CSGO{
		client: client,
	}
	client.GC.RegisterPacketHandler(c)
	time.Sleep(5 * time.Second)
	c.SendHello()
	c.SetPlaying(true)
	return c
}

func (c *CSGO) SendHello() {
	c.client.GC.Write(gamecoordinator.NewGCMsgProtobuf(AppId, uint32(protobuf.EGCBaseClientMsg_k_EMsgGCClientHello), &protobuf.CMsgClientHello{}))
}

func (c *CSGO) SetPlaying(playing bool) {
	if playing {
		c.client.GC.SetGamesPlayed(AppId)
	} else {
		c.client.GC.SetGamesPlayed()
	}
}

func (c *CSGO) InspectItem(typ string, owner string, assetid string, d string) {
	msg := &protobuf.CMsgGCCStrike15V2_Client2GCEconPreviewDataBlockRequest{
		ParamA: Param(assetid),
		ParamD: Param(d),
	}
	if typ == "S" {
		msg.ParamS = Param(owner)
	} else if typ == "M" {
		msg.ParamM = Param(owner)
	}
	data := gamecoordinator.NewGCMsgProtobuf(AppId, uint32(protobuf.ECsgoGCMsg_k_EMsgGCCStrike15_v2_Client2GCEconPreviewDataBlockRequest), msg)
	c.client.GC.Write(data)
}

func (c *CSGO) GetFloatvalue(value *uint32) string {
	fv := math.Float32frombits(uint32(*value))
	return fmt.Sprintf("%.25f", fv)
}

type (
	Sticker struct {
		Slot      *uint32  `json:"slot,omitempty"`
		StickerId *uint32  `json:"sticker_id,omitempty"`
		Wear      *float32 `json:"wear,omitempty"`
		Scale     *float32 `json:"scale,omitempty"`
		Rotation  *float32 `json:"rotation,omitempty"`
		TintId    *uint32  `json:"tint_id,omitempty"`
	}
	ItemInfo struct {
		AccountId          *uint32   `json:"accountid,omitempty"`
		ItemId             *uint64   `json:"itemid,omitempty"`
		DefIndex           *uint32   `json:"defindex,omitempty"`
		PaintIndex         *uint32   `json:"paintindex,omitempty"`
		Rarity             *uint32   `json:"rarity,omitempty"`
		Quality            *uint32   `json:"quality,omitempty"`
		PaintWear          *uint32   `json:"paintwear,omitempty"`
		PaintSeed          *uint32   `json:"paintseed,omitempty"`
		KillEaterScoreType *uint32   `json:"killeaterscoretype,omitempty"`
		KillEaterValue     *uint32   `json:"killeatervalue,omitempty"`
		CustomName         *string   `json:"customname,omitempty"`
		Stickers           []Sticker `json:"stickers,omitempty"`
		Inventory          *uint32   `json:"inventory,omitempty"`
		Origin             *uint32   `json:"origin,omitempty"`
		QuestId            *uint32   `json:"questid,omitempty"`
		DropReason         *uint32   `json:"dropreason,omitempty"`
		MusicIndex         *uint32   `json:"musicindex,omitempty"`
		FloatValue         string    `json:"floatvalue,omitempty"`
	}
	ClientReady struct{}
)

func (c *CSGO) HandleGCPacket(packet *gamecoordinator.GCPacket) {
	switch packet.MsgType {
	case uint32(protobuf.EGCBaseClientMsg_k_EMsgGCClientWelcome):
		c.handleWelcome(packet)
	case uint32(protobuf.EGCBaseClientMsg_k_EMsgGCClientConnectionStatus):
		// will return a status msg when the connection fails
		c.handleConnectionStatus(packet)
	case uint32(protobuf.ECsgoGCMsg_k_EMsgGCCStrike15_v2_MatchmakingGC2ClientHello):
		c.handleMatchmakingClientHello(packet)
	case uint32(protobuf.ECsgoGCMsg_k_EMsgGCCStrike15_v2_Client2GCEconPreviewDataBlockResponse):
		c.handleEconPreviewDataBlockResponse(packet)
	}
}

func (c *CSGO) handleWelcome(packet *gamecoordinator.GCPacket) {
	c.client.Emit(&ClientReady{})
}

func (c *CSGO) handleConnectionStatus(packet *gamecoordinator.GCPacket) {
	print("Connection failed... Retrying...\n")
	c.SendHello()
	c.SetPlaying(true)
}

func (c *CSGO) handleMatchmakingClientHello(packet *gamecoordinator.GCPacket) {
	data := &protobuf.CMsgGCCStrike15V2_MatchmakingGC2ClientHello{}
	packet.ReadProtoMsg(data)
	c.client.Emit(data)
}

func (c *CSGO) handleEconPreviewDataBlockResponse(packet *gamecoordinator.GCPacket) {
	data := &protobuf.CMsgGCCStrike15V2_Client2GCEconPreviewDataBlockResponse{}
	packet.ReadProtoMsg(data)
	dataInfo := data.GetIteminfo()
	itemInfo := &ItemInfo{
		AccountId:          dataInfo.Accountid,
		ItemId:             dataInfo.Itemid,
		DefIndex:           dataInfo.Defindex,
		PaintIndex:         dataInfo.Paintindex,
		Rarity:             dataInfo.Rarity,
		Quality:            dataInfo.Quality,
		PaintWear:          dataInfo.Paintwear,
		PaintSeed:          dataInfo.Paintseed,
		KillEaterScoreType: dataInfo.Killeaterscoretype,
		KillEaterValue:     dataInfo.Killeatervalue,
		CustomName:         dataInfo.Customname,
		Inventory:          dataInfo.Inventory,
		Origin:             dataInfo.Origin,
		QuestId:            dataInfo.Questid,
		DropReason:         dataInfo.Dropreason,
		MusicIndex:         dataInfo.Musicindex,
		FloatValue:         c.GetFloatvalue(dataInfo.Paintwear),
	}
	for _, sticker := range dataInfo.Stickers {
		itemInfo.Stickers = append(itemInfo.Stickers, Sticker{
			Slot:      sticker.Slot,
			StickerId: sticker.StickerId,
			Wear:      sticker.Wear,
			Scale:     sticker.Scale,
			Rotation:  sticker.Rotation,
			TintId:    sticker.TintId,
		})
	}
	c.client.Emit(itemInfo)
}
