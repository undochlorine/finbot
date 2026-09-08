package telegram

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
	"finbot/internal/text"
)

func menuCommands() []models.BotCommand {
	return []models.BotCommand{
		{Command: commandStart, Description: text.CmdDescStart},
		{Command: commandHelp, Description: text.CmdDescHelp},
		{Command: domain.CommandNewBank, Description: text.CmdDescNewBank},
		{Command: domain.CommandAdd, Description: text.CmdDescAdd},
		{Command: domain.CommandSpend, Description: text.CmdDescSpend},
		{Command: domain.CommandSet, Description: text.CmdDescSet},
		{Command: domain.CommandDelete, Description: text.CmdDescDelete},
		{Command: domain.CommandBank, Description: text.CmdDescBank},
		{Command: domain.CommandToggle, Description: text.CmdDescToggle},
		{Command: domain.CommandRename, Description: text.CmdDescRename},
		{Command: domain.CommandBanks, Description: text.CmdDescBanks},
		{Command: domain.CommandTotal, Description: text.CmdDescTotal},
		{Command: domain.CommandAll, Description: text.CmdDescAll},
		{Command: commandCancel, Description: text.CmdDescCancel},
		{Command: commandFeedback, Description: text.CmdDescFeedback},
	}
}

func (b *Bot) registerMenuCommands(ctx context.Context) error {
	ok, err := b.inner.SetMyCommands(ctx, &bot.SetMyCommandsParams{Commands: menuCommands()})
	if err != nil {
		return fmt.Errorf("setMyCommands: %w", err)
	}
	if !ok {
		return fmt.Errorf("setMyCommands returned false")
	}
	return nil
}
