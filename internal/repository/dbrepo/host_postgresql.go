package dbrepo

import (
	"context"
	"github.com/luksbutz/vigilate/internal/models"
	"time"
)

// InsertHost inserts a host into the database
func (m *postgresDBRepo) InsertHost(h models.Host) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `insert into hosts (
                   host_name, canonical_name, url, ip, ipv6, location,
                   os, active, created_at, updated_at)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) returning id`

	var newID int
	err := m.DB.QueryRowContext(ctx, stmt,
		h.HostName,
		h.CanonicalName,
		h.URL,
		h.IP,
		h.IPV6,
		h.Location,
		h.OS,
		h.Active,
		time.Now(),
		time.Now()).Scan(&newID)
	if err != nil {
		return 0, err
	}

	// add host services and set to inactive
	stmt = `
		insert into host_services (host_id, service_id, active, schedule_number, schedule_unit, created_at, updated_at, status)
		values ($1, 1, 0, 3, 'm', $2, $3, 'pending')
`

	_, err = m.DB.ExecContext(ctx, stmt, newID, time.Now(), time.Now())
	if err != nil {
		return newID, err
	}

	return newID, nil
}

// GetHostByID returns a host by id
func (m *postgresDBRepo) GetHostByID(id int) (models.Host, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		select
		    id, host_name, canonical_name, url, ip, ipv6, location, os, active, created_at, updated_at
		from hosts
		where id = $1
`

	var host models.Host
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&host.ID,
		&host.HostName,
		&host.CanonicalName,
		&host.URL,
		&host.IP,
		&host.IPV6,
		&host.Location,
		&host.OS,
		&host.Active,
		&host.CreatedAt,
		&host.UpdatedAt,
	)
	if err != nil {
		return host, err
	}

	// get all services for host
	query = `
		select hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number,
		       hs.schedule_unit, hs.last_check, hs.created_at, hs.updated_at, hs.status,
		       s.id, s.service_name, s.active, s.icon, s.created_at, s.updated_at
		from
		    host_services hs
			left join services s on s.id = hs.service_id
		where host_id = $1
`

	rows, err := m.DB.QueryContext(ctx, query, host.ID)
	if err != nil {
		return host, err
	}
	defer rows.Close()

	var hostServices []models.HostService

	for rows.Next() {
		var hs models.HostService
		err := rows.Scan(
			&hs.ID,
			&hs.HostID,
			&hs.ServiceID,
			&hs.Active,
			&hs.ScheduleNumber,
			&hs.ScheduleUnit,
			&hs.LastCheck,
			&hs.CreatedAt,
			&hs.UpdatedAt,
			&hs.Status,
			&hs.Service.ID,
			&hs.Service.ServiceName,
			&hs.Service.Active,
			&hs.Service.Icon,
			&hs.Service.UpdatedAt,
			&hs.Service.CreatedAt,
		)
		if err != nil {
			return host, err
		}
		hostServices = append(hostServices, hs)
	}

	host.HostServices = hostServices

	return host, rows.Err()
}

// UpdateHost updates a host
func (m *postgresDBRepo) UpdateHost(h models.Host) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `
update hosts set host_name = $1, canonical_name = $2, url = $3, ip = $4,
                 ipv6 = $5, location = $6, os = $7, active = $8, updated_at = $9 where id = $10`

	_, err := m.DB.ExecContext(ctx, stmt,
		h.HostName,
		h.CanonicalName,
		h.URL,
		h.IP,
		h.IPV6,
		h.Location,
		h.OS,
		h.Active,
		time.Now(),
		h.ID,
	)

	return err
}

// AllHosts returns a slice with all hosts
func (m *postgresDBRepo) AllHosts() ([]models.Host, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
	select id, host_name, canonical_name, url, ip, ipv6, location, os, active, created_at, updated_at
	from hosts
`

	var hosts []models.Host

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var h models.Host
		err = rows.Scan(
			&h.ID,
			&h.HostName,
			&h.CanonicalName,
			&h.URL,
			&h.IP,
			&h.IPV6,
			&h.Location,
			&h.OS,
			&h.Active,
			&h.CreatedAt,
			&h.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		serviceQuery := `
		select hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number,
		       hs.schedule_unit, hs.last_check, hs.created_at, hs.updated_at, hs.status,
		       s.id, s.service_name, s.active, s.icon, s.created_at, s.updated_at
		from host_services hs
		left join services s on s.id = hs.service_id
		where host_id = $1
`

		serviceRows, err := m.DB.QueryContext(ctx, serviceQuery, h.ID)
		if err != nil {
			return nil, err
		}

		var hostServices []models.HostService

		for serviceRows.Next() {
			var hs models.HostService
			err = serviceRows.Scan(
				&hs.ID,
				&hs.HostID,
				&hs.ServiceID,
				&hs.Active,
				&hs.ScheduleNumber,
				&hs.ScheduleUnit,
				&hs.LastCheck,
				&hs.CreatedAt,
				&hs.UpdatedAt,
				&hs.Status,
				&hs.Service.ID,
				&hs.Service.ServiceName,
				&hs.Service.Active,
				&hs.Service.Icon,
				&hs.Service.CreatedAt,
				&hs.Service.UpdatedAt,
			)
			if err != nil {
				return nil, err
			}
			hostServices = append(hostServices, hs)
		}

		if err := serviceRows.Err(); err != nil {
			return nil, err
		}

		serviceRows.Close()

		h.HostServices = hostServices

		hosts = append(hosts, h)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return hosts, nil
}

// UpdateHostServiceStatus updates the active status of a host service
func (m *postgresDBRepo) UpdateHostServiceStatus(hostID, serviceID, active int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `update host_services set active = $1 where host_id = $2 and service_id = $3`

	_, err := m.DB.ExecContext(ctx, stmt, active, hostID, serviceID)

	return err
}

// UpdateHostServiceStatus updates a host service in the database
func (m *postgresDBRepo) UpdateHostService(hs models.HostService) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `
		update host_services set
		        host_id = $1, service_id = $2, active = $3, schedule_number = $4, schedule_unit = $5,
			    last_check = $6, updated_at = $7, status = $8
		where id = $9

`

	_, err := m.DB.ExecContext(ctx, stmt,
		hs.HostID,
		hs.ServiceID,
		hs.Active,
		hs.ScheduleNumber,
		hs.ScheduleUnit,
		hs.LastCheck,
		hs.UpdatedAt,
		hs.Status,
		hs.ID,
	)

	return err
}

// GetAllServiceStatusCounts returns the count for all active services according to there status
func (m *postgresDBRepo) GetAllServiceStatusCounts() (int, int, int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var healthy, warning, problem, pending int

	query := `
select 
	(select count(id) from host_services where active = 1 and status = 'healthy') as healthy,
	(select count(id) from host_services where active = 1 and status = 'warning') as warning,
	(select count(id) from host_services where active = 1 and status = 'problem') as problem,
	(select count(id) from host_services where active = 1 and status = 'pending') as pending
`

	err := m.DB.QueryRowContext(ctx, query).Scan(&healthy, &warning, &problem, &pending)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	return healthy, warning, problem, pending, nil
}

func (m *postgresDBRepo) GetServicesByStatus(status string) ([]models.HostService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		select
			hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number,
			hs.schedule_unit, hs.last_check, hs.created_at, hs.updated_at, hs.status,
			h.host_name, s.service_name
		from
			host_services hs
			left join hosts h on (hs.host_id = h.id)
			left join services s on (hs.service_id = s.id)
		where
		    status = $1
			and hs.active = 1
		order by
		    host_name, service_name
`

	var services []models.HostService

	rows, err := m.DB.QueryContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var hs models.HostService
		err := rows.Scan(
			&hs.ID,
			&hs.HostID,
			&hs.ServiceID,
			&hs.Active,
			&hs.ScheduleNumber,
			&hs.ScheduleUnit,
			&hs.LastCheck,
			&hs.CreatedAt,
			&hs.UpdatedAt,
			&hs.Status,
			&hs.HostName,
			&hs.Service.ServiceName,
		)
		if err != nil {
			return nil, err
		}
		services = append(services, hs)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return services, err
}

func (m *postgresDBRepo) GetHostServiceByID(id int) (models.HostService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		select
		    hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number,
			hs.schedule_unit, hs.last_check, hs.created_at, hs.updated_at, hs.status,
			s.id, s.service_name, s.active, s.icon, s.created_at, s.updated_at
		from host_services hs
			left join services s on(hs.service_id = s.id)
		where
		    hs.id = $1
`

	var hs models.HostService

	row := m.DB.QueryRowContext(ctx, query, id)

	err := row.Scan(
		&hs.ID,
		&hs.HostID,
		&hs.ServiceID,
		&hs.Active,
		&hs.ScheduleNumber,
		&hs.ScheduleUnit,
		&hs.LastCheck,
		&hs.CreatedAt,
		&hs.UpdatedAt,
		&hs.Status,
		&hs.Service.ID,
		&hs.Service.ServiceName,
		&hs.Service.Active,
		&hs.Service.Icon,
		&hs.Service.CreatedAt,
		&hs.Service.UpdatedAt,
	)
	if err != nil {
		return hs, err
	}

	return hs, nil
}

func (m *postgresDBRepo) GetServicesToMonitor() ([]models.HostService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		select 
		    hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number,
			hs.schedule_unit, hs.last_check, hs.created_at, hs.updated_at, hs.status,
			s.id, s.service_name, s.active, s.icon, s.created_at, s.updated_at,
			h.host_name
		from host_services hs
			left join services s on(hs.service_id = s.id)
			left join hosts h on(h.id = hs.host_id)
		where
		    h.active = 1
			and hs.active = 1
`

	var services []models.HostService

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var hs models.HostService
		err := rows.Scan(
			&hs.ID,
			&hs.HostID,
			&hs.ServiceID,
			&hs.Active,
			&hs.ScheduleNumber,
			&hs.ScheduleUnit,
			&hs.LastCheck,
			&hs.CreatedAt,
			&hs.UpdatedAt,
			&hs.Status,
			&hs.Service.ID,
			&hs.Service.ServiceName,
			&hs.Service.Active,
			&hs.Service.Icon,
			&hs.Service.CreatedAt,
			&hs.Service.UpdatedAt,
			&hs.HostName,
		)
		if err != nil {
			return nil, err
		}
		services = append(services, hs)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return services, nil
}
