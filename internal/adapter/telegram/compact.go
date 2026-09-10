package telegram

import (
	"strings"

	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
)

func compact(jobs []pendingJob) []pendingJob {
	return dropReplacedEmptyWizards(collapseIdenticalRuns(jobs))
}

func collapseIdenticalRuns(jobs []pendingJob) []pendingJob {
	out := make([]pendingJob, 0, len(jobs))
	var prevKey string
	var prevOK bool
	for _, job := range jobs {
		key, ok := collapseKey(job)
		if ok && prevOK && key == prevKey {
			continue
		}
		out = append(out, job)
		prevKey, prevOK = key, ok
	}
	return out
}

func dropReplacedEmptyWizards(jobs []pendingJob) []pendingJob {
	for {
		out := make([]pendingJob, 0, len(jobs))
		dropped := false
		for i, job := range jobs {
			if i+1 < len(jobs) && isEmptyWizard(job) && isFlowStartSlash(jobs[i+1]) {
				dropped = true
				continue
			}
			out = append(out, job)
		}
		if !dropped {
			return out
		}
		jobs = out
	}
}

func slashCommand(update *models.Update) (cmd, payload string, ok bool) {
	if update == nil || update.Message == nil {
		return "", "", false
	}
	msg := update.Message.Text
	if !strings.HasPrefix(msg, "/") {
		return "", "", false
	}
	cmd, _, _ = strings.Cut(msg[1:], " ")
	cmd, _, _ = strings.Cut(cmd, "@")
	cmd = strings.ToLower(cmd)
	if cmd == "" {
		return "", "", false
	}
	return cmd, commandPayload(msg), true
}

func collapseKey(job pendingJob) (string, bool) {
	cmd, payload, ok := slashCommand(job.update)
	if !ok {
		return "", false
	}
	name := strings.ToLower(payload)
	switch {
	case isIdempotent(cmd, payload):
		return "id:" + cmd + "\x00" + name, true
	case payload == "" && isEmptyWizardCmd(cmd):
		return "empty:" + cmd, true
	case cmd == domain.CommandNewBank && payload != "":
		return "newbank:" + name, true
	case cmd == domain.CommandDelete && payload != "":
		return "delete:" + name, true
	default:
		return "", false
	}
}

func isIdempotent(cmd, payload string) bool {
	switch cmd {
	case commandHelp, commandStart, domain.CommandBanks, domain.CommandTotal, domain.CommandAll:
		return true
	case domain.CommandBank:
		return payload != ""
	default:
		return false
	}
}

func isEmptyWizardCmd(cmd string) bool {
	switch cmd {
	case domain.CommandNewBank, domain.CommandAdd, domain.CommandSpend, domain.CommandSet,
		domain.CommandDelete, domain.CommandToggle, domain.CommandBank, domain.CommandRename,
		domain.CommandTransfer, commandFeedback, commandCancel, commandLanguage:
		return true
	default:
		return false
	}
}

func isEmptyWizard(job pendingJob) bool {
	cmd, payload, ok := slashCommand(job.update)
	return ok && payload == "" && isEmptyWizardCmd(cmd)
}

func isFlowStartSlash(job pendingJob) bool {
	cmd, _, ok := slashCommand(job.update)
	if !ok {
		return false
	}
	switch cmd {
	case domain.CommandNewBank, domain.CommandAdd, domain.CommandSpend, domain.CommandSet,
		domain.CommandDelete, domain.CommandToggle, domain.CommandBank, domain.CommandRename,
		domain.CommandTransfer, commandFeedback, commandLanguage:
		return true
	default:
		return false
	}
}
