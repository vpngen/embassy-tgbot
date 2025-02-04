package tgbotapi

// ReactionType is the type of reaction.
// https://core.telegram.org/bots/api#reactiontype
type ReactionType struct {
	Type string `json:"type"` // “emoji”, “custom_emoji”, "paid"

	Emoji         string `json:"emoji,omitempty"`
	CustomEmojiID string `json:"custom_emoji_id,omitempty"`
}

// MessageReaction is a reaction to a message.
// https://core.telegram.org/bots/api#messagereactionupdated
type MessageReactionUpdated struct {
	Chat        *Chat          `json:"chat"`
	MessageID   int            `json:"message_id"`
	User        *User          `json:"user,omitempty"`
	ActionChat  *Chat          `json:"action_chat,omitempty"`
	Date        int            `json:"date"`
	OldReaction []ReactionType `json:"old_reaction"`
	NewReaction []ReactionType `json:"new_reaction"`
}
