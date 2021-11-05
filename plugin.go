package harukap

type HarukaPlugin interface {
	OnInit(e *HarukaAppEngine) error
}
