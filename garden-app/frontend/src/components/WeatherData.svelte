<script lang="ts">
    import type { WeatherData } from "../lib/waterScheduleClient";

    export let weatherData: WeatherData;
</script>

{#if weatherData.rain != null}
    Rain MM: {weatherData.rain.mm}<br />
    Rain Scale Factor: {weatherData.rain.scale_factor}<br />
{/if}
{#if weatherData.average_temperature != null}
    Average High Temp: {weatherData.average_temperature.celsius * 1.8 + 32}°F<br />
    Average High Temp Scale Factor: {weatherData.average_temperature.scale_factor}<br />
{/if}
