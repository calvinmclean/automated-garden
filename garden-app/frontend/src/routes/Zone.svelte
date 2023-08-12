<script lang="ts">
    import { onMount } from "svelte";
    import Zone from "../components/zones/Zone.svelte";
    import { getZone, type ZoneResponse, type GetZoneParams } from "../lib/zoneClient";
    import { zoneStore } from "../store";

    export let params: GetZoneParams;

    let zone: ZoneResponse;
    let loadingWeatherData = true;

    zoneStore.init(params.gardenID);
    zoneStore.subscribe((value) => {
        loadingWeatherData = value.loading;
        zone = value.zones.find((z) => z.id == params.zoneID);
    });
</script>

{#if zone}
    <Zone gardenID={params.gardenID} {zone} {loadingWeatherData} />
{/if}
