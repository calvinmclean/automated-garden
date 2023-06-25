//
//  EditZoneEditor.swift
//  Garden App
//
//  Created by Calvin McLean on 12/26/22.
//

import SwiftUI

struct EditZoneEditor: View {
    @EnvironmentObject var modelData: ModelData
    @State var zoneCount: Int = 0
    @State var editWaterSchedule: Bool = false
    
    @Environment(\.dismiss) var dismiss
    
    var garden: Garden
    var zone: Zone
    @Binding var newZone: UpdateZoneRequest
    
    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("Zone")) {
                    HStack {
                        Text("Name").frame(width: 100, alignment: .leading)
                        TextField("", text: $newZone.name)
                            .textFieldStyle(RoundedBorderTextFieldStyle())
                            .autocapitalization(.words)
                    }
                    HStack {
                        Text("Zone Position:")
                        Text("\(newZone.position)")
                        Stepper("", value: $newZone.position, in: 0...garden.maxZones - 1)
                    }
                }
                
//                Toggle(isOn: $editWaterSchedule) {
//                    Text("Edit Water Schedule")
//                }.onChange(of: editWaterSchedule) {_ in
//                    if (editWaterSchedule) {
//                        newZone.waterSchedule = zone.waterSchedule
//                    } else {
//                        newZone.waterSchedule = nil
//                    }
//                }
//                if (editWaterSchedule) {
//                    EditWaterSchedule(waterSchedule: Binding(
//                        get: {newZone.waterSchedule ?? zone.waterSchedule},
//                        set: {
//                            if newZone.waterSchedule == nil {
//                                newZone.waterSchedule = zone.waterSchedule
//                            }
//                            newZone.waterSchedule = $0
//                        }
//                    ))
//                }
                
                Section(header: Text("Details")) {
                    HStack {
                        Text("Description").frame(width: 100, alignment: .leading)
                        TextField("", text: Binding(
                            get: {newZone.details?.description ?? ""},
                            set: {
                                if newZone.details == nil {
                                    newZone.details = ZoneDetails()
                                }
                                newZone.details?.description = $0
                            }
                        ))
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                    }
                    HStack {
                        Text("Notes").frame(width: 100, alignment: .leading)
                        TextField("", text: Binding(
                            get: {newZone.details?.notes ?? ""},
                            set: {
                                if newZone.details == nil {
                                    newZone.details = ZoneDetails()
                                }
                                newZone.details?.notes = $0
                            }
                        ))
                            .textFieldStyle(RoundedBorderTextFieldStyle())
                    }
                }
                
                Section {
                    Button(action: {
                        modelData.zoneResource().updateZone(zone: zone, newZone: newZone)
                        dismiss()
                    }) {
                        Text("Submit")
                    }
                }
            }
        }
    }
}
