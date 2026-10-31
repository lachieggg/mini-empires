package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	train := flag.Duration("train", 0, "train the AI by self-play without a window for this long, then exit")
	flag.Parse()
	if *train > 0 {
		trainHeadless(*train)
		return
	}
	ebiten.SetWindowSize(ScreenW, ScreenH)
	ebiten.SetWindowTitle("Mini Empires")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	app := NewApp()
	defer app.trainer.Stop()
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}

func trainHeadless(d time.Duration) {
	t := NewTrainer(policyPath())
	t.Start()
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		time.Sleep(min(10*time.Second, time.Until(end)))
		s := t.Stats()
		fmt.Printf("games %5d (%.1f/s)  updates %4d  win vs scripted %3.0f%% (n=%d)  score %5.0f  length %s  entropy %.2f\n",
			s.Games, s.GamesPerS, s.Updates, s.EvalWin*100, s.EvalN, s.Score, clock(s.Duration), s.Last.Entropy)
	}
	t.Stop()
	fmt.Println("saved", policyPath())
}
