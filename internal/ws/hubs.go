package ws

type Hubs struct {
	Monitoring *MonitoringHub
	Student    *StudentHub
}

func NewHubs() *Hubs {
	return &Hubs{
		Monitoring: NewMonitoringHub(),
		Student:    NewStudentHub(),
	}
}
