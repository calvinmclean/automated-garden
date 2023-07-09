<script lang="ts">
    import {
        Badge,
        Card,
        CardBody,
        CardFooter,
        CardHeader,
        CardSubtitle,
        CardText,
        CardTitle,
        Icon,
        Spinner,
    } from "sveltestrap";
    import { fly } from "svelte/transition";
    import { location } from "svelte-spa-router";
    import type { WaterScheduleResponse } from "../../lib/waterScheduleClient";
    import WeatherData from "../WeatherData.svelte";
    import NextWater from "../NextWater.svelte";

    export let waterSchedule: WaterScheduleResponse;
    export let withLink = true;
    export let loadingWeatherData = false;
</script>

<div in:fly={{ x: 50, duration: 500 }} out:fly={{ x: -50, duration: 250 }}>
    <Card class=".col-lg-4" style="margin: 5%">
        {#if withLink}
            <a href="#{$location}/{waterSchedule.id}">
                <CardHeader>
                    <CardTitle>
                        {waterSchedule.name}
                        {#if loadingWeatherData}
                            <Spinner color={"success"} type="border" />
                        {/if}
                    </CardTitle>
                </CardHeader>
            </a>
        {:else}
            <CardHeader>
                <CardTitle>
                    {waterSchedule.name}
                    {#if loadingWeatherData}
                        <Spinner color={"success"} type="border" />
                    {/if}
                </CardTitle>
            </CardHeader>
        {/if}
        <CardBody>
            <CardSubtitle>{waterSchedule.id}</CardSubtitle>
            <CardText>
                {#if waterSchedule.end_date != null}
                    End Dated: {waterSchedule.end_date}
                    <Icon name="clock-fill" style="color: red" /><br />
                {/if}

                {#if waterSchedule.next_water != null}
                    <NextWater nextWater={waterSchedule.next_water} />
                {/if}

                {#if waterSchedule.weather_data != null}
                    <WeatherData weatherData={waterSchedule.weather_data} />
                {/if}
            </CardText>
        </CardBody>
        <CardFooter>
            {#if waterSchedule.end_date != null}
                <Badge color={"danger"}>End Dated</Badge>
            {/if}
        </CardFooter>
    </Card>
</div>
