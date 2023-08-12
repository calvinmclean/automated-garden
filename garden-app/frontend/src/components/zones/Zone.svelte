<script lang="ts">
    import { Accordion, AccordionItem, Button, Column, Col, FormGroup, Input, Label, Row, Spinner, Table } from "sveltestrap";
    import { waterZone, getZoneWaterHistory, type ZoneResponse, type WaterHistoryResponse } from "../../lib/zoneClient";
    import ZoneCard from "./ZoneCard.svelte";
    import { onMount } from "svelte";

    export let gardenID: string;
    export let zone: ZoneResponse;
    export let loadingWeatherData = false;

    let unit = zone.next_water.duration.includes("h") ? "m" : "s";
    let waterDurationValue = 1;

    let minutesToSeconds = (m: number): number => m * 60.0;

    function sendWaterRequest(event) {
        waterZone(gardenID, zone.id, unit == "s" ? waterDurationValue : minutesToSeconds(waterDurationValue));
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

    <FormGroup>
        <Row>
            <Col>
                <Input
                    type="range"
                    name="waterDurationRange"
                    id="waterDurationRange"
                    min={1}
                    max={120}
                    step={1}
                    bind:value={waterDurationValue}
                />
            </Col>
            <Col xs="1">
                <Input type="number" bind:value={waterDurationValue} min="0" />
            </Col>
            <Col>
                <Button on:click={sendWaterRequest} color={"primary"}>Water for {waterDurationValue}{unit}</Button>
            </Col>
        </Row>
    </FormGroup>

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
