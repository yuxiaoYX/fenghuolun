package store

import (
	"fmt"
	"time"

	"fenghuolun/internal/clock"
	"fenghuolun/internal/model/do"
	"fenghuolun/internal/model/entity"
	"fenghuolun/internal/neta"

	"github.com/gogf/gf/v2/database/gdb"
)

func (s *SQLite) SyncFills(bindingID string) error {
	if bindingID == "" {
		return nil
	}
	b := s.loadBinding(bindingID, "", true)
	if b == nil {
		return nil
	}
	for _, cand := range neta.DetectFills(b.Snapshots) {
		if err := s.upsertAutoFill(bindingID, cand); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLite) upsertAutoFill(bindingID string, cand neta.Fill) error {
	from := cand.FromFetched.Unix()
	to := cand.ToFetched.Unix()
	if from <= 0 || to <= 0 {
		return nil
	}
	var rows []entity.FillEvent
	err := s.db.Model("fill_event").Ctx(s.ctx()).
		Where("binding_id", bindingID).Where("kind", cand.Kind).
		Where("from_fetched", from).OrderAsc("created_at").Scan(&rows)
	if err != nil {
		return err
	}
	for _, row := range rows {
		// An ignored automatic event suppresses regeneration after the owner deletes it.
		if row.Status == "ignored" {
			return nil
		}
	}
	if len(rows) > 0 {
		row := rows[0]
		for _, candidate := range rows {
			if candidate.PaidCny != nil || candidate.Status == neta.FillRecorded {
				row = candidate
				break
			}
		}
		if row.Status == neta.FillRecorded || row.PaidCny != nil {
			// Keep a paid record's user-entered values, but remove duplicate drafts.
			for _, duplicate := range rows {
				if duplicate.Id != row.Id && duplicate.Status != "ignored" {
					_, _ = s.db.Model("fill_event").Ctx(s.ctx()).Where("id", duplicate.Id).Delete()
				}
			}
			return nil
		}
		_, err = s.db.Model("fill_event").Ctx(s.ctx()).Where("id", row.Id).Data(do.FillEvent{
			StartedAt:  cand.StartedAt.Unix(),
			FinishedAt: cand.FinishedAt.Unix(),
			OdoStart:   nvlFloat(cand.OdoStart),
			OdoEnd:     nvlFloat(cand.OdoEnd),
			SocStart:   nvlFloat(cand.SocStart),
			SocEnd:     nvlFloat(cand.SocEnd),
			FuelStart:  nvlFloat(cand.FuelStart),
			FuelEnd:    nvlFloat(cand.FuelEnd),
		}).Update()
		for _, duplicate := range rows {
			if duplicate.Id != row.Id {
				_, _ = s.db.Model("fill_event").Ctx(s.ctx()).Where("id", duplicate.Id).Delete()
			}
		}
		return err
	}
	id := NewSessionID()
	_, err = s.db.Model("fill_event").Ctx(s.ctx()).Data(fillData(id, bindingID, cand)).Insert()
	return err
}

func (s *SQLite) ListFills(bindingID string) ([]neta.Fill, error) {
	_ = s.SyncFills(bindingID)
	var rows []entity.FillEvent
	if err := s.db.Model("fill_event").Ctx(s.ctx()).Where("binding_id", bindingID).
		Where("status !=", "ignored").
		OrderDesc("finished_at").OrderDesc("created_at").Scan(&rows); err != nil {
		return nil, err
	}
	out := make([]neta.Fill, 0, len(rows))
	for _, r := range rows {
		out = append(out, neta.AnnotateFill(fillFromRow(r)))
	}
	return out, nil
}

func (s *SQLite) GetFill(bindingID, id string) *neta.Fill {
	if bindingID == "" || id == "" {
		return nil
	}
	var row entity.FillEvent
	if err := s.db.Model("fill_event").Ctx(s.ctx()).Where("id", id).Where("binding_id", bindingID).Where("status !=", "ignored").Scan(&row); err != nil || row.Id == "" {
		return nil
	}
	f := neta.AnnotateFill(fillFromRow(row))
	return &f
}

func (s *SQLite) SaveFill(bindingID string, in neta.Fill) (neta.Fill, error) {
	if bindingID == "" {
		return neta.Fill{}, fmt.Errorf("invalid_request: missing binding id")
	}
	if in.Kind != neta.FillCharge && in.Kind != neta.FillRefuel {
		return neta.Fill{}, fmt.Errorf("invalid_request: fill kind")
	}
	if err := checkFillNum(in.PaidCny, 0, 100000, "paid"); err != nil {
		return neta.Fill{}, err
	}
	if err := checkFillNum(in.EnergyKwh, 0.01, 200, "kwh"); err != nil {
		return neta.Fill{}, err
	}
	if err := checkFillNum(in.Liters, 0.01, 120, "liters"); err != nil {
		return neta.Fill{}, err
	}
	if err := checkFillNum(in.SocStart, 0, 100, "soc"); err != nil {
		return neta.Fill{}, err
	}
	if err := checkFillNum(in.SocEnd, 0, 100, "soc"); err != nil {
		return neta.Fill{}, err
	}
	if in.Source == "" {
		in.Source = neta.FillManual
	}
	startedProvided := !in.StartedAt.IsZero()
	finishedProvided := !in.FinishedAt.IsZero()
	if in.PaidCny != nil {
		in.Status = neta.FillRecorded
	} else {
		in.Status = neta.FillDraft
	}
	if in.FinishedAt.IsZero() {
		in.FinishedAt = clock.Of(time.Now().UTC())
	}
	if in.StartedAt.IsZero() {
		in.StartedAt = in.FinishedAt
	}
	if in.ID == "" {
		in.ID = NewSessionID()
		if _, err := s.db.Model("fill_event").Ctx(s.ctx()).Data(fillData(in.ID, bindingID, in)).Insert(); err != nil {
			return neta.Fill{}, err
		}
	} else {
		cur := s.GetFill(bindingID, in.ID)
		if cur == nil {
			return neta.Fill{}, fmt.Errorf("not_found")
		}
		if !startedProvided {
			in.StartedAt = cur.StartedAt
		}
		if !finishedProvided {
			in.FinishedAt = cur.FinishedAt
		}
		// Editing an auto-detected draft must keep its provenance. The request
		// body does not need to echo source just to add the receipt later.
		if in.Source == neta.FillManual && cur.Source == neta.FillAuto {
			in.Source = cur.Source
		}
		if in.FromFetched.IsZero() {
			in.FromFetched = cur.FromFetched
		}
		if in.ToFetched.IsZero() {
			in.ToFetched = cur.ToFetched
		}
		if in.OdoStart == nil {
			in.OdoStart = cur.OdoStart
		}
		if in.OdoEnd == nil {
			in.OdoEnd = cur.OdoEnd
		}
		if in.SocStart == nil {
			in.SocStart = cur.SocStart
		}
		if in.SocEnd == nil {
			in.SocEnd = cur.SocEnd
		}
		if in.FuelStart == nil {
			in.FuelStart = cur.FuelStart
		}
		if in.FuelEnd == nil {
			in.FuelEnd = cur.FuelEnd
		}
		if in.EnergyKwh == nil {
			in.EnergyKwh = cur.EnergyKwh
		}
		if in.Liters == nil {
			in.Liters = cur.Liters
		}
		if in.PaidCny == nil {
			in.PaidCny = cur.PaidCny
		}
		if in.Note == "" {
			in.Note = cur.Note
		}
		if in.PaidCny != nil {
			in.Status = neta.FillRecorded
		} else {
			in.Status = neta.FillDraft
		}
		_, err := s.db.Model("fill_event").Ctx(s.ctx()).Where("id", in.ID).Where("binding_id", bindingID).Data(do.FillEvent{
			Kind:       in.Kind,
			Status:     in.Status,
			StartedAt:  in.StartedAt.Unix(),
			FinishedAt: in.FinishedAt.Unix(),
			OdoStart:   nvlFloat(in.OdoStart),
			OdoEnd:     nvlFloat(in.OdoEnd),
			SocStart:   nvlFloat(in.SocStart),
			SocEnd:     nvlFloat(in.SocEnd),
			FuelStart:  nvlFloat(in.FuelStart),
			FuelEnd:    nvlFloat(in.FuelEnd),
			EnergyKwh:  nvlFloat(in.EnergyKwh),
			Liters:     nvlFloat(in.Liters),
			PaidCny:    nvlFloat(in.PaidCny),
			Note:       in.Note,
		}).Update()
		if err != nil {
			return neta.Fill{}, err
		}
	}
	got := s.GetFill(bindingID, in.ID)
	if got == nil {
		return neta.Fill{}, fmt.Errorf("not_found")
	}
	return *got, nil
}

func (s *SQLite) DeleteFill(bindingID, id string) error {
	if bindingID == "" || id == "" {
		return fmt.Errorf("not_found")
	}
	var row entity.FillEvent
	if err := s.db.Model("fill_event").Ctx(s.ctx()).Where("id", id).Where("binding_id", bindingID).Scan(&row); err != nil || row.Id == "" {
		return fmt.Errorf("not_found")
	}
	if row.Source == neta.FillAuto {
		res, err := s.db.Model("fill_event").Ctx(s.ctx()).Where("id", id).Where("binding_id", bindingID).Data(do.FillEvent{Status: "ignored"}).Update()
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return fmt.Errorf("not_found")
		}
		return nil
	}
	res, err := s.db.Model("fill_event").Ctx(s.ctx()).Where("id", id).Where("binding_id", bindingID).Delete()
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not_found")
	}
	return nil
}

func fillData(id, bindingID string, in neta.Fill) do.FillEvent {
	return do.FillEvent{
		Id:          id,
		BindingId:   bindingID,
		Kind:        in.Kind,
		Source:      in.Source,
		Status:      in.Status,
		FromFetched: in.FromFetched.Unix(),
		ToFetched:   in.ToFetched.Unix(),
		StartedAt:   in.StartedAt.Unix(),
		FinishedAt:  in.FinishedAt.Unix(),
		OdoStart:    nvlFloat(in.OdoStart),
		OdoEnd:      nvlFloat(in.OdoEnd),
		SocStart:    nvlFloat(in.SocStart),
		SocEnd:      nvlFloat(in.SocEnd),
		FuelStart:   nvlFloat(in.FuelStart),
		FuelEnd:     nvlFloat(in.FuelEnd),
		EnergyKwh:   nvlFloat(in.EnergyKwh),
		Liters:      nvlFloat(in.Liters),
		PaidCny:     nvlFloat(in.PaidCny),
		Note:        in.Note,
	}
}

func fillFromRow(r entity.FillEvent) neta.Fill {
	return neta.Fill{
		ID:          r.Id,
		Kind:        r.Kind,
		Source:      r.Source,
		Status:      r.Status,
		FromFetched: unixInstant(r.FromFetched),
		ToFetched:   unixInstant(r.ToFetched),
		StartedAt:   unixInstant(r.StartedAt),
		FinishedAt:  unixInstant(r.FinishedAt),
		OdoStart:    r.OdoStart,
		OdoEnd:      r.OdoEnd,
		SocStart:    r.SocStart,
		SocEnd:      r.SocEnd,
		FuelStart:   r.FuelStart,
		FuelEnd:     r.FuelEnd,
		EnergyKwh:   r.EnergyKwh,
		Liters:      r.Liters,
		PaidCny:     r.PaidCny,
		Note:        r.Note,
	}
}

func unixInstant(u int64) clock.Instant {
	if u <= 0 {
		return clock.Instant{}
	}
	return clock.Of(time.Unix(u, 0).UTC())
}

func nvlFloat(v *float64) interface{} {
	if v == nil {
		return gdb.Raw("NULL")
	}
	return *v
}

func checkFillNum(v *float64, lo, hi float64, name string) error {
	if v == nil {
		return nil
	}
	if *v < lo || *v > hi {
		return fmt.Errorf("invalid_request: fill %s", name)
	}
	return nil
}
