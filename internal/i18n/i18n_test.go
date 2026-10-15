package i18n

import (
	"strings"
	"testing"

	"moneykeeper/internal/models"
)

func TestReminderText(t *testing.T) {
	max := int64(180000)
	base := models.Schedule{Name: "Аренда", Currency: "GEL", AmountKind: models.AmountExact, AmountMinor: 150000}
	tests := []struct {
		name string
		r    models.Reminder
		want []string
	}{
		{"today exact", models.Reminder{Schedule: base}, []string{"Сегодня платёж «Аренда»: 1\u00a0500,00\u00a0₾."}},
		{"tomorrow", models.Reminder{Schedule: base, Lead: 1}, []string{"Завтра платёж «Аренда»"}},
		{"in three days", models.Reminder{Schedule: base, Lead: 3}, []string{"Через 3 дн. платёж"}},
		{"approx", models.Reminder{Schedule: func() models.Schedule { s := base; s.AmountKind = models.AmountApprox; return s }()}, []string{"около 1\u00a0500,00\u00a0₾"}},
		{"range", models.Reminder{Schedule: func() models.Schedule {
			s := base
			s.AmountKind, s.AmountMaxMinor = models.AmountRange, &max
			return s
		}()}, []string{"от 1\u00a0500,00\u00a0₾ до 1\u00a0800,00\u00a0₾"}},
		{"explains what to do", models.Reminder{Schedule: base}, []string{"внесите трату", "пропустите"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReminderText(tt.r)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("%q does not contain %q", got, w)
				}
			}
		})
	}
}
