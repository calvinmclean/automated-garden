<script lang="ts">
    import type { ZoneResponse } from "../../lib/zoneClient";

    export let zone: ZoneResponse;
</script>

{#if zone}
    <h1>{zone.name}</h1>
{/if}
