//
//  ZoneWaterHistoryView.swift
//  Garden App
//
//  Created by Calvin McLean on 12/5/21.
//

import SwiftUI

struct ZoneWaterHistoryView: View {
    var waterHistory: WaterHistoryResponse
    
    var body: some View {
        List {
            Section("Details") {
                DetailHStack(key: "Count", value: String(waterHistory.count))
                DetailHStack(key: "Average", value: waterHistory.average)
                DetailHStack(key: "Total", value: waterHistory.total)
            }
            if let history = waterHistory.history {
                ForEach(history, id: \.self.recordTime) { event in
                    Section {
                        DetailHStack(key: "Duration", value: event.duration)
                        DetailHStack(key: "Time", value: event.recordTime.formattedWithTime)
                    }
                }
            }
        }
        .navigationTitle(Text("Water History"))
    }
}
