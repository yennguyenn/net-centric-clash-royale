package handlers

import (
	"sync"
	"time"

	"tcr_project/models"
)

const (
	ManaRegenRate = 1           // mana regen mỗi giây
	MaxMana       = 10          // giới hạn tối đa
	TickDuration  = time.Second // 1 giây
)

// StartManaRegeneration sẽ chạy liên tục và hồi mana cho người chơi
func StartManaRegeneration(players []*models.Player, mutex *sync.Mutex) {
	go func() {
		for {
			time.Sleep(TickDuration)
			mutex.Lock()
			for _, p := range players {
				if p.Mana < MaxMana {
					p.Mana += ManaRegenRate
					if p.Mana > MaxMana {
						p.Mana = MaxMana
					}
				}
			}
			mutex.Unlock()
		}
	}()
}
