package web

import (
	"net/http"

	"moneykeeper/internal/models"
	"moneykeeper/internal/web/apidto"
	"moneykeeper/pkg/apptime"
)

func scheduleDTO(s models.Schedule) apidto.ScheduleDTO {
	return apidto.ScheduleDTO{
		ID: s.ID, Name: s.Name, CategoryID: s.CategoryID, AccountID: s.AccountID, Currency: string(s.Currency),
		AmountKind: string(s.AmountKind), AmountMinor: s.AmountMinor, AmountMaxMinor: s.AmountMaxMinor,
		EveryMonths: s.EveryMonths, Days: s.Days, StartMonth: s.StartMonth.String(), WeekendShift: string(s.Shift), RemindBefore: s.RemindBefore,
	}
}

func (h *handler) newSchedule(w http.ResponseWriter, req apidto.ScheduleRequest) (models.NewSchedule, bool) {
	in := models.NewSchedule{
		Name: req.Name, CategoryID: req.CategoryID, AccountID: req.AccountID, Currency: req.Currency,
		AmountKind: models.AmountKind(req.AmountKind), AmountMinor: req.AmountMinor, AmountMaxMinor: req.AmountMaxMinor,
		EveryMonths: req.EveryMonths, Days: req.Days, Shift: models.WeekendShift(req.WeekendShift), RemindBefore: req.RemindBefore,
	}
	if req.StartMonth != nil {
		m, err := apptime.ParseMonth(*req.StartMonth)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidDate)
			return in, false
		}
		in.StartMonth = &m
	}
	return in, true
}

func (h *handler) listSchedules(w http.ResponseWriter, r *http.Request, u models.User) {
	list, err := h.svc.ListSchedules(r.Context(), u)
	if err != nil {
		h.fail(w, err)
		return
	}
	out := apidto.SchedulesResponse{Schedules: make([]apidto.ScheduleDTO, 0, len(list))}
	for _, s := range list {
		out.Schedules = append(out.Schedules, scheduleDTO(s))
	}
	h.writeJSON(w, http.StatusOK, out)
}

func (h *handler) createSchedule(w http.ResponseWriter, r *http.Request, u models.User) {
	var req apidto.ScheduleRequest
	if !h.decode(w, r, &req) {
		return
	}
	in, ok := h.newSchedule(w, req)
	if !ok {
		return
	}
	s, err := h.svc.CreateSchedule(r.Context(), u, in)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, scheduleDTO(s))
}

func (h *handler) updateSchedule(w http.ResponseWriter, r *http.Request, u models.User) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var req apidto.ScheduleRequest
	if !h.decode(w, r, &req) {
		return
	}
	in, ok := h.newSchedule(w, req)
	if !ok {
		return
	}
	s, err := h.svc.UpdateSchedule(r.Context(), u, id, in)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scheduleDTO(s))
}

func (h *handler) deleteSchedule(w http.ResponseWriter, r *http.Request, u models.User) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteSchedule(r.Context(), u, id); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// occurrences отдаёт платежи за период (параметры from и to, по умолчанию от месяца назад до двух месяцев вперёд).
func (h *handler) occurrences(w http.ResponseWriter, r *http.Request, u models.User) {
	var from, to *apptime.Day
	for name, dst := range map[string]**apptime.Day{"from": &from, "to": &to} {
		if raw := r.URL.Query().Get(name); raw != "" {
			d, err := apptime.ParseDay(raw)
			if err != nil {
				h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidDate)
				return
			}
			*dst = &d
		}
	}
	list, err := h.svc.Occurrences(r.Context(), u, from, to)
	if err != nil {
		h.fail(w, err)
		return
	}
	out := apidto.OccurrencesResponse{Occurrences: make([]apidto.OccurrenceDTO, 0, len(list))}
	for _, o := range list {
		out.Occurrences = append(out.Occurrences, apidto.OccurrenceDTO{Schedule: scheduleDTO(o.Schedule), DueOn: o.DueOn.String(), Status: string(o.Status), TransactionID: o.TransactionID})
	}
	h.writeJSON(w, http.StatusOK, out)
}

// skipOccurrence пропускает платёж на дату: он перестаёт быть просроченным и не напоминает.
func (h *handler) skipOccurrence(w http.ResponseWriter, r *http.Request, u models.User) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var req apidto.SkipRequest
	if !h.decode(w, r, &req) {
		return
	}
	due, err := apptime.ParseDay(req.DueOn)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidDate)
		return
	}
	if err := h.svc.SkipOccurrence(r.Context(), u, id, due); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
