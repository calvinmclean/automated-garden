<script lang="ts">
    import Zone from "../components/zones/Zone.svelte";
    import { type ZoneResponse, type GetZoneParams } from "../lib/zoneClient";
    import { createZoneStore } from "../store";

    export let params: GetZoneParams;

    let zone: ZoneResponse;
    let loadingWeatherData = true;

    let zoneStoreInstance = createZoneStore(params.gardenID);
    zoneStoreInstance.subscribe((zd) => {
        loadingWeatherData = zd.loading;
        zone = zd.zones.find((z: ZoneResponse) => z.id == params.zoneID);
    });
</script>

{#if zone}
    <Zone gardenID={params.gardenID} {zone} {loadingWeatherData} />
{/if}
