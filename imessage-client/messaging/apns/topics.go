package apns

// Topic represents an APNS topic for iMessage services.
type Topic string

// APNS topics used by iMessage and related services.
const (
	TopicMadrid                       Topic = "com.apple.madrid"
	TopicAlloySMS                     Topic = "com.apple.private.alloy.sms"
	TopicAlloyGelato                  Topic = "com.apple.private.alloy.gelato"
	TopicAlloyBiz                     Topic = "com.apple.private.alloy.biz"
	TopicAlloySafetyMonitor           Topic = "com.apple.private.alloy.safetymonitor"
	TopicAlloySafetyMonitorOwnAccount Topic = "com.apple.private.alloy.safetymonitor.ownaccount"
	TopicAlloyGamecenteriMessage      Topic = "com.apple.private.alloy.gamecenter.imessage"
	TopicAlloyAskTo                   Topic = "com.apple.private.alloy.askto"
	TopicIDS                          Topic = "com.apple.private.ids"
)

// APNS courier connection parameters.
const (
	CourierHostCount = 50
	CourierHostname  = "courier.push.apple.com"
	CourierPort      = 5223
)
