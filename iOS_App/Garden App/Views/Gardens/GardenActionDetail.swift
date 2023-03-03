//
//  GardenActionDetail.swift
//  Garden App
//
//  Created by Calvin McLean on 12/21/21.
//

import SwiftUI

struct GardenActionDetail: View {
    @EnvironmentObject var modelData: ModelData
    let garden: Garden
    
    @State var stopAll: Bool = false
    @State var delayAmount: Int = 30
    @State var lightDelayState: Bool = false
    
    var body: some View {
        List {
            Section("Actions") {
                VStack {
                    Toggle(isOn: $stopAll) {
                        Text("Clear Watering Queue")
                    }
                    HStack {
                        Button(action: {
                            print("Stop watering button tapped for \(garden.name)")
                            modelData.gardenResource().stopWatering(garden: garden, all: stopAll)
                        }) {
                            Label { Text("Stop Watering") } icon: { Image(systemName: "exclamationmark.octagon.fill") }
                            .frame(minWidth: 150, maxWidth: 150)
                        }
                        .buttonStyle(ActionButtonStyle(bgColor: .red))
                        .controlSize(.large)
                        Spacer()
                    }
                }.padding(10)
                
                if (garden.lightSchedule != nil) {
                    VStack {
                        HStack {
                            Button(action: {
                                print("Light ON button tapped for \(garden.name)")
                                modelData.gardenResource().toggleLight(garden: garden, state: "ON")
                            }) {
                                Label { Text("ON") } icon: { Image(systemName: "sunrise") }
                            }
                            .buttonStyle(ActionButtonStyle(bgColor: .yellow))
                            .controlSize(.regular)
                            
                            Button(action: {
                                print("Light OFF button tapped for \(garden.name)")
                                modelData.gardenResource().toggleLight(garden: garden, state: "OFF")
                            }) {
                                Label { Text("OFF") } icon: { Image(systemName: "sunset") }
                            }
                            .buttonStyle(ActionButtonStyle(bgColor: .gray))
                            .controlSize(.regular)
                        }
                    }.padding(10)
                    
                    VStack {
                        Toggle(isOn: $lightDelayState) {
                            Text("Desired State")
                        }
                        Stepper(value: $delayAmount, in: 5...120, step: 5) {
                            Button(action: {
                                print("Delay light button tapped for \(garden.name)")
                                let state = lightDelayState ? "ON" : "OFF"
                                modelData.gardenResource().delayLight(garden: garden, minutes: delayAmount, state: state)
                            }) {
                                Label { Text("Delay Light (\(delayAmount)m)") } icon: { Image(systemName: "cloud.sun") }
                                .frame(minWidth: 150, maxWidth: 150)
                            }
                            .buttonStyle(ActionButtonStyle(bgColor: .gray))
                            .controlSize(.large)
                        }
                        
                    }.padding(10)
                }
            }
        }
        .listStyle(InsetGroupedListStyle())
        .navigationTitle(garden.name + " Actions")
    }
}
