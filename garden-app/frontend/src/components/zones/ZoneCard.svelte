<script lang="ts">
    import {
        Badge,
        Button,
        ButtonDropdown,
        Card,
        CardBody,
        CardFooter,
        CardHeader,
        CardText,
        CardTitle,
        Col,
        Collapse,
        DropdownToggle,
        DropdownMenu,
        DropdownItem,
        Icon,
        Popover,
        Row,
        Spinner,
        NavItem,
    } from "sveltestrap";
    import { fly } from "svelte/transition";
    import { location } from "svelte-spa-router";
    import { endDateZone, restoreZone, type ZoneResponse } from "../../lib/zoneClient";
    import type { WaterScheduleResponse } from "../../lib/waterScheduleClient";
    import WaterScheduleCard from "../waterSchedules/WaterScheduleCard.svelte";
    import { waterScheduleStore } from "../../store";
    import WeatherData from "../WeatherData.svelte";
    import NextWater from "../NextWater.svelte";

    export let gardenID: string;
    export let zone: ZoneResponse;
    export let withLink = true;
    export let loadingWeatherData = false;

    let waterScheduleCollapseIsOpen: boolean = false;

    let nextWaterSchedule: WaterScheduleResponse;

    waterScheduleStore.subscribe((value) => {
        if (zone.next_water != null) {
            nextWaterSchedule = waterScheduleStore.getByID(value, zone.next_water.water_schedule_id);
        }
    });

    function deleteZone(event) {
        zone.end_date = Date.now().toLocaleString();
        endDateZone(gardenID, zone.id);
    }

    function onRestore(event) {
        zone.end_date = null;
        restoreZone(gardenID, zone.id);
    }
</script>

<div in:fly={{ x: 50, duration: 500 }} out:fly={{ x: -50, duration: 250 }}>
    <Card class=".col-lg-4" style="margin: 5%">
        {#if withLink}
            <a href="#{$location}/zones/{zone.id}">
                <CardHeader>
                    <CardTitle>
                        {zone.name}
                        {#if loadingWeatherData}
                            <Spinner color={"success"} type="border" />
                        {/if}
                    </CardTitle>
                </CardHeader>
            </a>
        {:else}
            <CardHeader>
                <CardTitle>
                    {zone.name}
                    {#if loadingWeatherData}
                        <Spinner color={"success"} type="border" />
                    {/if}
                </CardTitle>
            </CardHeader>
        {/if}
        <CardBody>
            {#if zone.next_water != null}
                <Card on:click={() => (waterScheduleCollapseIsOpen = !waterScheduleCollapseIsOpen)}>
                    <CardBody>
                        This Zone will be watered for {zone.next_water.duration} at {zone.next_water.time}
                        <Icon name="cloud-drizzle" style="color: blue" />
                    </CardBody>
                </Card>

                {#if nextWaterSchedule != null}
                    <Collapse isOpen={waterScheduleCollapseIsOpen}>
                        <Card body>
                            <WaterScheduleCard waterSchedule={nextWaterSchedule} {loadingWeatherData} withLink={true} />
                        </Card>
                    </Collapse>
                {/if}
            {/if}

            <!-- TODO:
                - Add details/links to other water schedules
                - Show Zone details -->

            <CardText>
                <ButtonDropdown>
                    <DropdownToggle color="danger" caret>
                        <Icon name="trash" />
                    </DropdownToggle>
                    <DropdownMenu>
                        <DropdownItem on:click={deleteZone}>
                            {#if zone.end_date == null}
                                Confirm Delete
                            {:else}
                                Permanently Delete
                            {/if}
                        </DropdownItem>
                    </DropdownMenu>
                </ButtonDropdown>

                {#if zone.end_date != null}
                    <div>
                        <Button color="primary" on:click={onRestore}>
                            <Icon name="arrow-clockwise" />
                        </Button>
                    </div>
                {/if}
            </CardText>
        </CardBody>
        <CardFooter>
            {#if zone.end_date != null}
                <Badge color={"danger"}>End Dated</Badge>
            {/if}

            <Row>
                <Col class="offset-sm-6">
                    <Icon name="info-circle" id={`info-${zone.id}`} />
                    <Popover trigger="hover" target={`info-${zone.id}`} placement="left" title="Zone Info">
                        ID: {zone.id}<br />
                        Position: {zone.position}<br />
                        Created At: {zone.created_at}<br />
                        WaterScheduleIDs:<br />
                        <ul>
                            {#each zone.water_schedule_ids as wsID}
                                <li>{wsID}</li>
                            {/each}
                        </ul>

                        {#if zone.end_date != null}
                            End Dated: {zone.end_date}<br />
                        {/if}
                    </Popover>
                </Col>
            </Row>
        </CardFooter>
    </Card>
</div>
