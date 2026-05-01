package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/units"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	"github.com/calvinmclean/babyapi/extensions"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

const (
	waterScheduleBasePath   = "/water_schedules"
	waterScheduleIDLogField = "water_schedule_id"
)

// WaterSchedulesAPI provides and API for interacting with WaterSchedules
type WaterSchedulesAPI struct {
	*babyapi.API[*pkg.WaterSchedule]

	storageClient *storage.Client
	worker        *worker.Worker
}

func NewWaterSchedulesAPI() *WaterSchedulesAPI {
	api := &WaterSchedulesAPI{}

	api.API = babyapi.NewAPI[*pkg.WaterSchedule]("WaterSchedules", waterScheduleBasePath, func() *pkg.WaterSchedule { return &pkg.WaterSchedule{} })

	api.SetResponseWrapper(func(ws *pkg.WaterSchedule) render.Renderer {
		return api.NewWaterScheduleResponse(ws)
	})
	api.SetSearchResponseWrapper(func(waterSchedules []*pkg.WaterSchedule) render.Renderer {
		resp := AllWaterSchedulesResponse{ResourceList: babyapi.ResourceList[*WaterScheduleResponse]{}}

		for _, w := range waterSchedules {
			resp.ResourceList.Items = append(resp.ResourceList.Items, api.NewWaterScheduleResponse(w))
		}

		return resp
	})

	api.SetOnCreateOrUpdate(api.onCreateOrUpdate)

	api.SetBeforeDelete(func(_ http.ResponseWriter, r *http.Request) *babyapi.ErrResponse {
		id := api.GetIDParam(r)

		// Unable to delete WaterSchedule with associated Zones
		zones, err := api.storageClient.GetZonesUsingWaterSchedule(id)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("unable to get Zones using WaterSchedule: %w", err))
		}
		if numZones := len(zones); numZones > 0 {
			return babyapi.ErrInvalidRequest(fmt.Errorf("unable to end-date WaterSchedule with %d Zones", numZones))
		}

		return nil
	})

	api.SetAfterDelete(func(_ http.ResponseWriter, r *http.Request) *babyapi.ErrResponse {
		logger := babyapi.GetLoggerFromContext(r.Context())
		id := api.GetIDParam(r)

		// Remove scheduled WaterActions
		logger.Info("removing scheduled WaterActions for WaterSchedule")
		err := api.worker.RemoveJobsByID(id)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("unable to remove scheduled WaterActions: %w", err))
		}

		return nil
	})

	api.AddCustomRoute(http.MethodGet, "/components", babyapi.Handler(func(_ http.ResponseWriter, r *http.Request) render.Renderer {
		switch r.URL.Query().Get("type") {
		case "create_modal":
			return api.waterScheduleModalRenderer(r, &pkg.WaterSchedule{
				ID: NewID(),
			})
		default:
			return babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	api.AddCustomIDRoute(http.MethodGet, "/components", api.GetRequestedResourceAndDo(func(_ http.ResponseWriter, r *http.Request, ws *pkg.WaterSchedule) (render.Renderer, *babyapi.ErrResponse) {
		switch r.URL.Query().Get("type") {
		case "edit_modal":
			return api.waterScheduleModalRenderer(r, ws), nil
		case "detail_modal":
			return waterScheduleDetailModalTemplate.Renderer(ws), nil
		default:
			return nil, babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	// Scaling example endpoint for previewing weather-based scaling
	api.AddCustomRoute(http.MethodPost, "/scaling_example", babyapi.Handler(api.scalingExample))

	api.ApplyExtension(extensions.HTMX[*pkg.WaterSchedule]{})

	api.EnableMCP(babyapi.MCPPermRead)

	return api
}

func (api *WaterSchedulesAPI) waterScheduleModalRenderer(r *http.Request, ws *pkg.WaterSchedule) render.Renderer {
	notificationClients, err := api.storageClient.NotificationClientConfigs.Search(r.Context(), "", nil)
	if err != nil {
		return babyapi.InternalServerError(fmt.Errorf("error getting all notification clients to create water schedule modal: %w", err))
	}

	slices.SortFunc(notificationClients, func(nc1 *notifications.Client, nc2 *notifications.Client) int {
		return strings.Compare(nc1.Name, nc2.Name)
	})

	weatherClients, err := api.storageClient.WeatherClientConfigs.Search(r.Context(), "", nil)
	if err != nil {
		return babyapi.InternalServerError(fmt.Errorf("error getting all weather clients to create water schedule modal: %w", err))
	}

	slices.SortFunc(weatherClients, func(wc1 *weather.Config, wc2 *weather.Config) int {
		return strings.Compare(wc1.Name, wc2.Name)
	})

	return waterScheduleModalTemplate.Renderer(struct {
		*pkg.WaterSchedule
		NotificationClients []*notifications.Client
		WeatherClients      []*weather.Config
	}{ws, notificationClients, weatherClients})
}

func (api *WaterSchedulesAPI) setup(storageClient *storage.Client, worker *worker.Worker) error {
	api.storageClient = storageClient
	api.worker = worker

	api.SetStorage(api.storageClient.WaterSchedules)

	// Initialize WaterActions for each WaterSchedule from the storage client
	allWaterSchedules, err := api.storageClient.WaterSchedules.Search(context.Background(), "", nil)
	if err != nil {
		return fmt.Errorf("unable to get WaterSchedules: %v", err)
	}
	for _, ws := range allWaterSchedules {
		if ws.EndDated() {
			continue
		}
		err = api.worker.ScheduleWaterAction(ws)
		if err != nil {
			return fmt.Errorf("unable to add WaterAction for WaterSchedule %v: %v", ws.ID, err)
		}
	}

	return nil
}

func (api *WaterSchedulesAPI) onCreateOrUpdate(_ http.ResponseWriter, r *http.Request, ws *pkg.WaterSchedule) *babyapi.ErrResponse {
	// Validate the new WaterSchedule.WeatherControl
	if ws.WeatherControl != nil {
		err := api.weatherClientsExist(r.Context(), ws)
		if err != nil {
			if errors.Is(err, babyapi.ErrNotFound) {
				return babyapi.ErrInvalidRequest(fmt.Errorf("unable to get WeatherClients for WaterSchedule: %w", err))
			}
			return babyapi.InternalServerError(err)
		}

		// Convert imperial values to metric if user is using imperial units
		if getUnitsFromRequest(r) == "imperial" {
			if ws.WeatherControl.Rain != nil {
				// Convert rain input range (inches to mm)
				if ws.WeatherControl.Rain.InputMin != nil {
					*ws.WeatherControl.Rain.InputMin = units.InchesToMm(*ws.WeatherControl.Rain.InputMin)
				}
				if ws.WeatherControl.Rain.InputMax != nil {
					*ws.WeatherControl.Rain.InputMax = units.InchesToMm(*ws.WeatherControl.Rain.InputMax)
				}
			}
			if ws.WeatherControl.Temperature != nil {
				// Convert temperature input range (°F to °C)
				if ws.WeatherControl.Temperature.InputMin != nil {
					*ws.WeatherControl.Temperature.InputMin = units.FahrenheitToCelsius(*ws.WeatherControl.Temperature.InputMin)
				}
				if ws.WeatherControl.Temperature.InputMax != nil {
					*ws.WeatherControl.Temperature.InputMax = units.FahrenheitToCelsius(*ws.WeatherControl.Temperature.InputMax)
				}
			}
		}

		err = pkg.ValidateWeatherControl(ws.WeatherControl)
		if err != nil {
			return babyapi.ErrInvalidRequest(fmt.Errorf("invalid WaterSchedule.WeatherControl after patching: %w", err))
		}
	}

	if !ws.EndDated() {
		err := api.worker.ResetWaterSchedule(ws)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("unable to update/reset WaterSchedule: %w", err))
		}
	}

	return nil
}

func (api *WaterSchedulesAPI) weatherClientsExist(ctx context.Context, ws *pkg.WaterSchedule) error {
	if ws.HasTemperatureControl() {
		err := api.weatherClientExists(ctx, ws.WeatherControl.Temperature.ClientID)
		if err != nil {
			return fmt.Errorf("error getting client for TemperatureControl: %w", err)
		}
	}

	if ws.HasRainControl() {
		err := api.weatherClientExists(ctx, ws.WeatherControl.Rain.ClientID)
		if err != nil {
			return fmt.Errorf("error getting client for RainControl: %w", err)
		}
	}

	return nil
}

func (api *WaterSchedulesAPI) weatherClientExists(ctx context.Context, id xid.ID) error {
	_, err := api.storageClient.WeatherClientConfigs.Get(ctx, id.String())
	if err != nil {
		return fmt.Errorf("error getting WeatherClient with ID %q: %w", id, err)
	}
	return nil
}

// scalingExample handles POST requests to preview how watering duration will scale based on weather configuration
// nolint:gosec // Request body size is limited by babyapi middleware
func (api *WaterSchedulesAPI) scalingExample(_ http.ResponseWriter, r *http.Request) render.Renderer {
	// Parse form data from the request
	if err := r.ParseForm(); err != nil {
		return babyapi.ErrInvalidRequest(fmt.Errorf("error parsing form data: %w", err))
	}

	userUnits := getUnitsFromRequest(r)
	isImperial := userUnits == "imperial"

	// Parse base duration and interval
	baseDuration := parseFormDuration(r, "Duration", 0)
	interval := parseFormDuration(r, "Interval", 24*time.Hour)

	response := ScalingExampleResponse{
		BaseDuration: baseDuration.Round(time.Second).String(),
	}

	// Parse ET scaling configuration
	etCanopyDiameter := parseFormFloat(r, "WeatherControl.Evapotranspiration.CanopyDiameterFeet")
	etSpecies := r.FormValue("WeatherControl.Evapotranspiration.Species")
	etFlowRate := parseFormFloat(r, "WeatherControl.Evapotranspiration.FlowRateGPH")
	etClientID := r.FormValue("WeatherControl.Evapotranspiration.ClientID")

	// If ET scaling is configured, calculate the ET-based duration
	effectiveBaseDuration := baseDuration
	if etCanopyDiameter != nil && etFlowRate != nil && etClientID != "" && etSpecies != "" {
		response.ETConfigured = true

		// Create a temporary ET config
		etConfig := &weather.EvapotranspirationScaler{
			CanopyDiameterFeet: float32(*etCanopyDiameter),
			Species:            weather.Species(etSpecies),
			FlowRateGPH:        float32(*etFlowRate),
		}

		// Calculate ET-based duration
		etDuration, etValue := api.calculateETDuration(r.Context(), etClientID, etConfig, interval, isImperial)
		response.ETDuration = etDuration.Round(time.Second).String()
		response.ETValue = etValue
		if etDuration > 0 {
			effectiveBaseDuration = etDuration
		}
	}

	// Parse rain scaling configuration
	rainInputMin := parseFormFloat(r, "WeatherControl.Rain.InputMin")
	rainInputMax := parseFormFloat(r, "WeatherControl.Rain.InputMax")
	rainFactorMin := parseFormFloat(r, "WeatherControl.Rain.FactorMin")
	rainFactorMax := parseFormFloat(r, "WeatherControl.Rain.FactorMax")
	rainInterpolation := r.FormValue("WeatherControl.Rain.Interpolation")

	// If rain scaling is configured, generate examples
	if rainInputMin != nil && rainInputMax != nil && rainFactorMin != nil && rainFactorMax != nil {
		// Convert imperial to metric if needed (form values are in user units)
		if isImperial {
			*rainInputMin = units.InchesToMm(*rainInputMin)
			*rainInputMax = units.InchesToMm(*rainInputMax)
		}

		scaler := &weather.WeatherScaler{
			Interpolation: weather.InterpolationMode(rainInterpolation),
			InputMin:      rainInputMin,
			InputMax:      rainInputMax,
			FactorMin:     rainFactorMin,
			FactorMax:     rainFactorMax,
		}

		response.RainExamples = generateScalingExamples(scaler, effectiveBaseDuration, isImperial, true)
	}

	// Parse temperature scaling configuration
	tempInputMin := parseFormFloat(r, "WeatherControl.Temperature.InputMin")
	tempInputMax := parseFormFloat(r, "WeatherControl.Temperature.InputMax")
	tempFactorMin := parseFormFloat(r, "WeatherControl.Temperature.FactorMin")
	tempFactorMax := parseFormFloat(r, "WeatherControl.Temperature.FactorMax")
	tempInterpolation := r.FormValue("WeatherControl.Temperature.Interpolation")

	// If temperature scaling is configured, generate examples
	if tempInputMin != nil && tempInputMax != nil && tempFactorMin != nil && tempFactorMax != nil {
		// Convert imperial to metric if needed (form values are in user units)
		if isImperial {
			*tempInputMin = units.FahrenheitToCelsius(*tempInputMin)
			*tempInputMax = units.FahrenheitToCelsius(*tempInputMax)
		}

		scaler := &weather.WeatherScaler{
			Interpolation: weather.InterpolationMode(tempInterpolation),
			InputMin:      tempInputMin,
			InputMax:      tempInputMax,
			FactorMin:     tempFactorMin,
			FactorMax:     tempFactorMax,
		}

		response.TemperatureExamples = generateScalingExamples(scaler, effectiveBaseDuration, isImperial, false)
	}

	return response
}

// calculateETDuration fetches ET data and calculates watering duration based on citrus tree formula
// Returns the duration and ET value (in user units for display)
func (api *WaterSchedulesAPI) calculateETDuration(ctx context.Context, etClientID string, etConfig *weather.EvapotranspirationScaler, interval time.Duration, isImperial bool) (time.Duration, float32) {
	clientID, err := xid.FromString(etClientID)
	if err != nil {
		return 0, 0
	}

	weatherClient, err := api.storageClient.GetWeatherClient(clientID)
	if err != nil {
		return 0, 0
	}

	// Check if client supports ET
	etProvider, ok := weatherClient.(weather.ETProvider)
	if !ok {
		return 0, 0
	}

	// Fetch average ET over the configured interval
	avgET, err := etProvider.GetAverageEvapotranspiration(ctx, interval)
	if err != nil {
		return 0, 0
	}

	// For imperial units, convert the display value to inches
	etValue := avgET
	if isImperial {
		etValue = avgET * 0.0393701 // mm to inches
	}

	// Calculate duration using the configured interval
	etDuration, err := worker.CalculateETDuration(etConfig, avgET, interval, time.Now())
	if err != nil {
		return 0, etValue
	}

	return etDuration, etValue
}

// parseFormFloat parses a float64 from form data
func parseFormFloat(r *http.Request, name string) *float64 {
	valueStr := r.FormValue(name)
	if valueStr == "" {
		return nil
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return nil
	}
	return &value
}

// parseFormDuration parses a time.Duration from form data
// Returns the default value if parsing fails or field is empty
func parseFormDuration(r *http.Request, name string, defaultValue time.Duration) time.Duration {
	valueStr := r.FormValue(name)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// generateScalingExamples creates 5 sample points across the input range
func generateScalingExamples(scaler *weather.WeatherScaler, baseDuration time.Duration, isImperial bool, isRain bool) []ScalingExamplePoint {
	if scaler.InputMin == nil || scaler.InputMax == nil {
		return nil
	}

	inputMin := *scaler.InputMin
	inputMax := *scaler.InputMax
	inputRange := inputMax - inputMin

	// Generate 5 sample points: min, 25%, 50%, 75%, max
	percentages := []float64{0.0, 0.25, 0.5, 0.75, 1.0}
	examples := make([]ScalingExamplePoint, len(percentages))

	for i, pct := range percentages {
		inputValue := inputMin + (inputRange * pct)
		scaleFactor := scaler.Scale(inputValue)

		// Convert to user units for display
		displayValue := inputValue
		unit := "mm"
		if isRain {
			if isImperial {
				displayValue = units.MmToInches(inputValue)
				unit = "in"
			}
		} else {
			if isImperial {
				displayValue = units.CelsiusToFahrenheit(inputValue)
				unit = "°F"
			} else {
				unit = "°C"
			}
		}

		examples[i] = ScalingExamplePoint{
			InputValue:  displayValue,
			InputUnit:   unit,
			ScaleFactor: scaleFactor,
		}

		// Calculate resulting duration if base duration is provided
		if baseDuration > 0 {
			resultDuration := time.Duration(float64(baseDuration) * scaleFactor)
			examples[i].Duration = formatDurationShort(resultDuration)
		}
	}

	return examples
}

// formatDurationShort formats a duration in a short, readable format
func formatDurationShort(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	// Round to nearest second to avoid sub-second precision
	d = d.Round(time.Second)

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	if minutes > 0 {
		if seconds > 0 {
			return fmt.Sprintf("%dm%ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", seconds)
}
