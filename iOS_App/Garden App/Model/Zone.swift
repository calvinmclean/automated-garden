//
//  Zone.swift
//  Garden App
//
//  Created by Calvin McLean on 5/8/22.
//

import Foundation

class Zone: Identifiable, Codable {
    var name: String = ""
    var details: ZoneDetails? = ZoneDetails()
    var id: String = ""
    var position: Int = 0
    var createdAt: Date = Date()
    var endDate: Date? = nil
    var skipCount: Int? = 0
    var waterScheduleIDs: Array<String> = []
    var nextWaterDetails: NextWaterDetails? = nil
    var links: Array<Link> = []
    var history: WaterHistoryResponse? = nil
    
    static let `default` = Zone()
    
    enum SortBy {
        case position
        case name
        case createdAt
//        case nextWatering
    }
    
    static var decoder: JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = JSONDecoder.DateDecodingStrategy.customISO8601
        return decoder
    }
    
    static var encoder: JSONEncoder {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = JSONEncoder.DateEncodingStrategy.customISO8601
        return encoder
    }
    
    func getLink(rel: String) -> String? {
        return self.links.first { l in
            l.rel == rel
        }?.href
    }
    
    func isEndDated() -> Bool {
        if let endDate = self.endDate {
            return endDate <= Date()
        }
        return false
    }
    
    func isLessThan(other: Zone, sortBy: SortBy) -> Bool {
        switch (self.isEndDated(), other.isEndDated(), sortBy) {
        case (true, true, .position), (false, false, .position):
            return self.position < other.position
        case (true, true, .name), (false, false, .name):
            return self.name < other.name
        case (true, true, .createdAt), (false, false, .createdAt):
            return self.createdAt < other.createdAt
//        case (true, true, .nextWatering), (false, false, .nextWatering):
//            return self.nextWaterTime ?? Date() < other.nextWaterTime ?? Date()
        case (true, false, _):
            return false
        case (false, true, _):
            return true
        }
    }
    
    enum CodingKeys: String, CodingKey {
        case name, id, links, position, details
        case createdAt = "created_at"
        case endDate = "end_date"
        case skipCount = "skip_count"
        case nextWaterDetails = "next_water"
        case waterScheduleIDs = "water_schedule_ids"
    }
}

struct ZoneDetails: Codable, Equatable {
    var description: String? = nil
    var notes: String? = nil
    
    enum CodingKeys: String, CodingKey {
        case description, notes
    }
}


struct WaterSchedule: Codable, Equatable {
    var duration: String = ""
    var interval: String = ""
    var startTime: Date = Date()
    var weatherControl: WeatherControl? = nil
    
    enum CodingKeys: String, CodingKey {
        case interval, duration
        case startTime = "start_time"
        case weatherControl = "weather_control"
    }
}

struct WeatherControl: Codable, Equatable {
    var rain: ScaleControl? = nil
    var temperature: ScaleControl? = nil
    var soilMoisture: SoilMoistureControl? = nil
    
    enum CodingKeys: String, CodingKey {
        case rain = "rain_control"
        case temperature = "temperature_control"
        case soilMoisture = "moisture_control"
    }
}

struct ScaleControl: Codable, Equatable {
    var baselineValue: Float32 = 0
    var factor: Float32 = 0
    var range: Float32 = 0
    
    enum CodingKeys: String, CodingKey {
        case factor, range
        case baselineValue = "baseline_value"
    }
}

struct SoilMoistureControl: Codable, Equatable {
    var minimumMoisture: Int = 0
    
    enum CodingKeys: String, CodingKey {
        case minimumMoisture = "minimum_moisture"
    }
}

struct NextWaterDetails: Codable {
    var time: Date? = nil
    var duration: String? = nil
    var waterScheduleID: String? = nil
    var message: String? = nil
    
    enum CodingKeys: String, CodingKey {
        case time, duration, message
        case waterScheduleID = "water_schedule_id"
    }
}

struct ListOfZones: Codable {
    let zones: [Zone]
}

struct Zones<T: Codable>: Codable {
    let zones: [T]
}
