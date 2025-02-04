package main

import tgbotapi "github.com/vpngen/embassy-tgbot/telegram-bot-api"

var Reactions = []string{
	"👍", "👎", "❤", "🔥", "🥰", "👏", "😁", "🤔", "🤯", "😱",
	"🤬", "😢", "🎉", "🤩", "🤮", "💩", "🙏", "👌", "🕊", "🤡",
	"🥱", "🥴", "😍", "🐳", "❤‍🔥", "🌚", "🌭", "💯", "🤣", "⚡",
	"🍌", "🏆", "💔", "🤨", "😐", "🍓", "🍾", "💋", "🖕", "😈",
	"😴", "😭", "🤓", "👻", "👨‍💻", "👀", "🎃", "🙈", "😇", "😨",
	"🤝", "✍", "🤗", "🫡", "🎅", "🎄", "☃", "💅", "🤪", "🗿",
	"🆒", "💘", "🙉", "🦄", "😘", "💊", "🙊", "😎", "👾", "🤷‍♂",
	"🤷", "🤷‍♀", "😡",
}

func AddedReactions(mru *tgbotapi.MessageReactionUpdated) []tgbotapi.ReactionType {
	added := make([]tgbotapi.ReactionType, 0)

	for _, newReaction := range mru.NewReaction {
		found := false
		for _, oldReaction := range mru.OldReaction {
			if newReaction == oldReaction {
				found = true

				break
			}
		}
		if !found {
			added = append(added, newReaction)
		}
	}

	return added
}
