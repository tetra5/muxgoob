package birthdays

import (
	"math/rand"
	"time"
	"strconv"
	"log"
	"regexp"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"github.com/tucnak/telebot"
	"github.com/bearbin/go-age"

	"github.com/focusshifter/muxgoob/registry"
)

type BirthdaysPlugin struct {
}

type BirthdayNotify struct {
	ID int `storm:"id,increment"`
	Username string `storm:"index"`
	Year int
}

var db *storm.DB
var rng *rand.Rand
var birthdays map[string]time.Time

func init() {
	registry.RegisterPlugin(&BirthdaysPlugin{})
}

func (p *BirthdaysPlugin) Start(sharedDb *storm.DB) {
	db = sharedDb
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	birthdays = make(map[string]time.Time)

	loc := registry.Config.TimeLoc

	for username, birthday := range registry.Config.Birthdays {
		t, _ := time.ParseInLocation("2006-01-02", birthday, loc)
		birthdays[username] = t
	}
}

func (p *BirthdaysPlugin) Process(message telebot.Message) {
	todaysBirthday(message)
	nextBirthday(message)
}

func nextBirthday(message telebot.Message) {
	bot := registry.Bot
	loc := registry.Config.TimeLoc

	birthdayExp := regexp.MustCompile(`(?i)^\!(др|birthda(y|ys))$`)

	switch {
		case birthdayExp.MatchString(message.Text):
			cur := time.Now().In(loc)
			curDay := cur.YearDay()

			diff := 365
			curDiff := 365
			curBirthday := ""
			curUsername := ""

			for username, birthday := range birthdays {
				diff = birthday.YearDay() - curDay
				if diff > 0 {
					if diff == curDiff {
						curUsername += ", @" + username
					} else if diff < curDiff {
						curDiff = diff
						curUsername = username	
						curBirthday = birthday.Format("01.02")
					}
				}
			}

			bot.SendMessage(message.Chat, "Prepare the 🎂 for @" + curUsername + " on " + curBirthday, &telebot.SendOptions{})
	}
}

func todaysBirthday(message telebot.Message) {
	bot := registry.Bot
	loc := registry.Config.TimeLoc

	cur := time.Now().In(loc)

	for username, birthday := range birthdays {
		if birthday.YearDay() == cur.YearDay() && notMentioned(username, birthday.Year(), message) {
			age := strconv.Itoa(age.AgeAt(birthday, cur));
			bot.SendMessage(message.Chat, "Hooray! 🎉 @" + username + " is turning " + age + "! 🎂", &telebot.SendOptions{})
		}
	}
}

func notMentioned(username string, year int, message telebot.Message) bool {
	chat := db.From(strconv.FormatInt(message.Chat.ID, 10))

	count, _ := chat.Select(q.And(q.Eq("Username", username), q.Eq("Year", year))).Count(&BirthdayNotify{})

	if count > 0 {
		return false
	}

	log.Println("Brithday: notify " + username)

	newNotify := BirthdayNotify{Username: username, Year: year}
	chat.Save(&newNotify)

	return true
}
