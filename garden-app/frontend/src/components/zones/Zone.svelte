<script lang="ts">
    import {
        Accordion,
        AccordionItem,
        Button,
        Column,
        FormGroup,
        Input,
        Label,
        Spinner,
        Table,
    } from "sveltestrap";
    import type {
        ZoneResponse,
        WaterHistoryResponse,
    } from "../../lib/zoneClient";
    import { waterZone, getZoneWaterHistory } from "../../lib/zoneClient";
    import ZoneCard from "./ZoneCard.svelte";
    import { onMount } from "svelte";

    export let gardenID: string;
    export let zone: ZoneResponse;
    export let loadingWeatherData = false;

    let minutes = 1;

    function sendWaterRequest(event) {
        waterZone(gardenID, zone.id, minutes);
    }

    let history: WaterHistoryResponse;
    let rangeDays: number = 15;
    let limit: number = 10;

    function zoneWaterHistoryRequest() {
        getZoneWaterHistory(gardenID, zone.id, rangeDays, limit)
            .then((response) => response.data)
            .then((data) => {
                history = data;
            });
    }

    onMount(async () => {
        zoneWaterHistoryRequest();
    });
</script>

{#if zone}
    <ZoneCard {gardenID} {zone} {loadingWeatherData} withLink={false} />

    <div>
        <input type="number" bind:value={minutes} min="0" /> minutes
        <Button on:click={sendWaterRequest} color={"primary"}>Water!</Button>
    </div>

    {#if history}
        <br />
        <Accordion flush>
            <AccordionItem header="Watering History">
                <FormGroup>
                    <Label for="waterHistoryDateRange">
                        Range: {rangeDays} Days
                    </Label>
                    <Input
                        type="range"
                        name="range"
                        id="waterHistoryDateRange"
                        min={1}
                        max={30}
                        step={1}
                        placeholder="Range placeholder"
                        bind:value={rangeDays}
                        on:change={zoneWaterHistoryRequest}
                    />

                    <Label for="waterHistoryLimit">
                        Limit: {limit} results
                    </Label>
                    <Input
                        type="range"
                        name="limit"
                        id="waterHistoryLimit"
                        min={1}
                        max={100}
                        step={1}
                        placeholder="Limit placeholder"
                        bind:value={limit}
                        on:change={zoneWaterHistoryRequest}
                    />
                </FormGroup>

                <Table striped borderless rows={history.history} let:row>
                    <Column header="Duration">
                        {row.duration}
                    </Column>
                    <Column header="Time">
                        {row.record_time}
                    </Column>
                </Table>
            </AccordionItem>
        </Accordion>
    {:else}
        <Spinner color={"success"} type="border" />
    {/if}
{/if}
