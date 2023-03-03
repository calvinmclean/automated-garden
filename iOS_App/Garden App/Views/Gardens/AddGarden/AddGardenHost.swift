//
//  AddGardenHost.swift
//  Garden App
//
//  Created by Calvin McLean on 11/23/22.
//

import SwiftUI

struct AddGardenHost: View {
    @EnvironmentObject var modelData: ModelData
    @State private var newGarden: CreateGardenRequest
    
    init() {
        self.newGarden = CreateGardenRequest()
    }
    
    
    var body: some View {
        AddGardenEditor(newGarden: $newGarden)
            .onAppear {
                newGarden = CreateGardenRequest()
            }
            .onDisappear {
                modelData.fetchGardens()
            }
    }
}
