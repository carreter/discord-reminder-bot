package discordreminderbot

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Sugared zap logger.
var sugar = zap.S()

// A Message represents the contents of a Reminder.
type Message struct {
	ChannelID   string
	PingUserIDs []string
	FromUserID  string
	Name        string
	Description string
}

// A MessageDispatcherFunc takes a Message and sends it appropriately.
type MessageDispatcherFunc func(context.Context, Message) error

// A Reminder is a schedulable Message.
type Reminder interface {
	ScheduleSend(context.Context, time.Time, MessageDispatcherFunc)
}

// A OneShotReminder is only sent once.
type OneShotReminder struct {
	gorm.Model
	Occurrence time.Time
	Message    Message
}

// ScheduleSend calls the dispatcherFunc at the reminder's occurrence and
func (r OneShotReminder) ScheduleSend(ctx context.Context, dispatcherFunc MessageDispatcherFunc) {
	go func() {
		select {
		case <-ctx.Done():
			sugar.Warnw("context cancelled before one-shot reminder could be sent", "occurrence", r.Occurrence, "message", r.Message)
		case <-time.After(time.Until(r.Occurrence)):
		}
	}()
}

type RepeatReminder struct {
	gorm.Model
	NextOcurrence  time.Time
	CronExpression string `gorm:"index,sort:asc"`
	Message        Message
}

type Scheduler struct {
	db        *gorm.DB
	pollEvery time.Duration
}

func NewScheduler(db *gorm.DB, pollEvery time.Duration) *Scheduler {
	return &Scheduler{
		db:        db,
		pollEvery: pollEvery,
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	sugar.Infow("starting scheduler", "pollEvery", s.pollEvery)

	window := s.pollEvery + time.Millisecond

	ticker := time.Tick(s.pollEvery)
	for {
		select {
		case <-ticker:
			var oneShotReminders []OneShotReminder
			err := s.db.Where("ocurrence < ?", time.Now().Add(window)).Find(&oneShotReminders).Error
			if err != nil {
				sugar.Errorw("failed to get one-shot reminders", "err", err)
				return err
			}

			var repeatReminders []RepeatReminder
			err = s.db.Where("ocurrence < ?", time.Now().Add(window)).Find(&repeat).Error

		case <-ctx.Done():
			sugar.Infow("context cancelled, shutting down scheduler", "err", ctx.Err())
			return ctx.Err()
		}
	}
}
