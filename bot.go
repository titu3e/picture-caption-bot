package main

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/golang/freetype/truetype"
)

type Bot struct {
	api     *tgbotapi.BotAPI
	cfg     *Config
	updates tgbotapi.UpdatesChannel
	errors  chan error
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	debug   bool

	whitelist map[int64]struct{}
	blacklist map[int64]struct{}

	font *truetype.Font
}

func New(cfg *Config) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, err
	}

	if cfg.Workers <= 0 {
		return nil, errors.New("number of workers must be positive")
	}

	api.Debug = cfg.Debug

	if cfg.Debug {
		log.Print("Running in debug mode")
	}

	fontData, err := ioutil.ReadFile(cfg.Font)
	if err != nil {
		return nil, err
	}

	font, err := truetype.Parse(fontData)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	var whitelist map[int64]struct{}
	var blacklist map[int64]struct{}

	if cfg.Whitelist != nil {
		whitelist = make(map[int64]struct{})
		for _, x := range cfg.Whitelist {
			whitelist[x] = struct{}{}
		}
	}

	if cfg.Blacklist != nil {
		blacklist = make(map[int64]struct{})
		for _, x := range cfg.Blacklist {
			blacklist[x] = struct{}{}
		}
	}

	initRand()

	return &Bot{
		api:     api,
		cfg:     cfg,
		updates: nil,
		ctx:     ctx,
		cancel:  cancel,
		wg:      sync.WaitGroup{},
		debug:   cfg.Debug,

		whitelist: whitelist,
		blacklist: blacklist,

		font: font,
	}, nil
}

func initRand() {
	var b [8]byte
	_, err := cryptorand.Read(b[:])
	if err != nil {
		panic("cannot seed math/rand package with cryptographically secure random number generator")
	}
	rand.Seed(int64(binary.LittleEndian.Uint64(b[:])))
}

func (bot *Bot) Start(ctx context.Context) (chan error, error) {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates, err := bot.api.GetUpdatesChan(updateConfig)
	if err != nil {
		return nil, err
	}

	bot.updates = updates
	bot.errors = make(chan error, bot.cfg.Workers)

	for i := 0; i < bot.cfg.Workers; i++ {
		go bot.worker()
	}

	done := make(chan error)

	go func() {
		// Block until one of two events occur
		select {
		// External termination
		case <-ctx.Done():
			// Do nothing
		// Error in one of the workers
		case err := <-bot.errors:
			// Propagate that error
			done <- err
		}

		// Stop all the workers
		bot.cancel()
		// Wait for them to join
		bot.wg.Wait()
		// Close the updates channel
		bot.api.StopReceivingUpdates()
		// Close the errors channel
		close(bot.errors)
		// Signal to external listener that we are done
		close(done)
	}()

	return done, nil
}

func (bot *Bot) worker() {
	bot.wg.Add(1)
	defer bot.wg.Done()

loop:
	for {
		select {
		case update := <-bot.updates:
			err := bot.processUpdate(update)
			if err != nil {
				bot.errors <- err
				break loop
			}
		case <-bot.ctx.Done():
			break loop
		}
	}
}

func (bot *Bot) processUpdate(update tgbotapi.Update) error {
	if update.Message == nil {
		bot.logDebug("Message is missing")
		return nil
	}

	msg := update.Message
	chat := msg.Chat
	fromID := chat.ID

	if !bot.isAllowed(fromID) {
		bot.logDebug("From id is not allowed: %v", fromID)
		return nil
	}

	if msg.Photo == nil {
		bot.logDebug("No photo")
		return nil
	}

	if chat.IsGroup() {
		if !bot.cfg.Group.Enabled {
			return nil
		}

		if bot.cfg.Group.ActivationPhrase != "" && bot.cfg.Group.ActivationPhrase != msg.Caption {
			if rand.Float64() > bot.cfg.Group.ActivationProbability {
				return nil
			}
		}
	}

	// Get photo with maximum width
	maxWidth := 0
	fileID := ""
	for _, photo := range *update.Message.Photo {
		if photo.Width > maxWidth {
			maxWidth = photo.Width
			fileID = photo.FileID
		}
	}

	// Get photo URL from file id
	file, err := bot.api.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return err
	}
	fileURL := file.Link(bot.cfg.Token)

	bot.logDebug("Downloading file: %s", fileURL)

	// Download photo into memory
	resp, err := http.Get(fileURL)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Decode image
	img, format, err := image.Decode(resp.Body)
	if err != nil {
		return err
	}

	bot.logDebug("Decoded image of format %s", format)

	phrase := bot.cfg.Phrases[rand.Intn(len(bot.cfg.Phrases))]
	out, err := drawCaption(img, bot.font, phrase)
	if err != nil {
		return err
	}

	bot.logDebug("Wrote string")

	// Encode output image as JPEG and write it to buffer
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, out, &jpeg.Options{Quality: 85})
	if err != nil {
		return err
	}

	bot.logDebug("Encoded JPEG")

	// Send image
	_, err = bot.api.Send(tgbotapi.NewPhotoUpload(
		fromID,
		tgbotapi.FileReader{
			Name:   "output.jpeg",
			Reader: buf,
			Size:   int64(buf.Len()),
		},
	))

	bot.logDebug("Sent photo")

	return err
}

func (bot *Bot) isAllowed(id int64) bool {
	if bot.blacklist != nil {
		if _, ok := bot.blacklist[id]; ok {
			return false
		}
	}

	if bot.whitelist != nil {
		_, ok := bot.whitelist[id]
		return ok
	}

	return true
}

func (bot *Bot) logDebug(format string, v ...interface{}) {
	if bot.debug {
		log.Printf(format, v...)
	}
}
