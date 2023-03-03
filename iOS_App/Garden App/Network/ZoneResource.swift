//
//  ZoneResource.swift
//  Garden App
//
//  Created by Calvin McLean on 6/13/21.
//

import Foundation

struct WaterAction: Codable {
    var duration: Int
    var ignoreMoisture: Bool
    
    enum CodingKeys: String, CodingKey {
        case duration
        case ignoreMoisture = "ignore_moisture"
    }
}

struct ZoneAction: Codable {
    var water: WaterAction
}

struct WaterScheduleStartTime: Codable {
    var startTime: Date = Date()
    
    enum CodingKeys: String, CodingKey {
        case startTime = "start_time"
    }
}

struct DelayWateringRequestBody: Codable {
    var waterSchedule: WaterScheduleStartTime = WaterScheduleStartTime()
    
    static var encoder: JSONEncoder {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = JSONEncoder.DateEncodingStrategy.customISO8601
        return encoder
    }
    
    enum CodingKeys: String, CodingKey {
        case waterSchedule = "water_schedule"
    }
}

struct CreateZoneRequest: Codable {
    var name: String = ""
    var position: Int = 0
    var details: ZoneDetails? = ZoneDetails()
    var waterSchedule: WaterSchedule = WaterSchedule()
    
    static var encoder: JSONEncoder {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = JSONEncoder.DateEncodingStrategy.customISO8601
        return encoder
    }
    
    enum CodingKeys: String, CodingKey {
        case name, position, details
        case waterSchedule = "water_schedule"
    }
}

class UpdateZoneRequest: Identifiable, Codable {
    var name: String = ""
    var details: ZoneDetails? = nil
    var moisture: Float? = nil
    var position: Int = 0
    var waterSchedule: WaterSchedule? = nil
    
    init(position: Int) {
        self.position = position
    }
    
    static let `default` = UpdateZoneRequest(position: 0)
    
    enum CodingKeys: String, CodingKey {
        case name, position, details
        case waterSchedule = "water_schedule"
    }
}

struct WaterHistory: Codable {
    var duration: String = ""
    var recordTime: Date = Date()
    
    static var decoder: JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = JSONDecoder.DateDecodingStrategy.customISO8601
        return decoder
    }
    
    enum CodingKeys: String, CodingKey {
        case duration
        case recordTime = "record_time"
    }
}

struct WaterHistoryResponse: Codable {
    var history: [WaterHistory]? = []
    var count: Int = 0
    var average: String = ""
    var total: String = ""
}

class ZoneResource {
    var url: String

    init(url: String) {
        self.url = url
    }

    func getZones(garden: Garden, showEndDated: Bool = false, withCompletion completion: @escaping ([Zone]?) -> Void) {
        guard let url = URL(string: "\(self.url)\(garden.zones.href)?end_dated=\(showEndDated)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        let task = URLSession.shared.dataTask(with: request) { (data, response, error) in
            if let data = data {
                do {
                    let zones = try Zone.decoder.decode(ListOfZones.self, from: data)
                    DispatchQueue.main.async { completion(zones.zones) }
                } catch let jsonErr {
                    print("JSON Error: \(jsonErr)")
//                    print("DATA: \(String(data: data, encoding: String.Encoding.utf8))")
                    DispatchQueue.main.async { completion(nil) }
                }
            }
        }
        task.resume()
    }

    func getZone(zoneID: String, withCompletion completion: @escaping (Zone?) -> Void) {
        let url = URL(string: "\(self.url)/zones/\(zoneID)")
        guard let requestUrl = url else { fatalError() }
        var request = URLRequest(url: requestUrl)
        request.httpMethod = "GET"
        let task = URLSession.shared.dataTask(with: request) { (data, response, error) in
            if let data = data {
                do {
                    let zone = try Zone.decoder.decode(Zone.self, from: data)
                    DispatchQueue.main.async { completion(zone) }
                } catch let jsonErr {
                    print("JSON Error: \(jsonErr)")
                    DispatchQueue.main.async { completion(nil) }
                }
            }
        }
        task.resume()
    }

    func waterZone(zone: Zone, duration: Int, ignoreMoisture: Bool) {
        guard let path = zone.getLink(rel: "action") else { fatalError() }
        guard let url = URL(string: "\(self.url)\(path)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        let requestBody = ZoneAction(water: WaterAction(duration: duration, ignoreMoisture: ignoreMoisture))
        let jsonData = try! JSONEncoder().encode(requestBody)
        request.httpBody = jsonData
        let task = URLSession.shared.dataTask(with: request)
        task.resume()
    }

    func delayWatering(zone: Zone, days: Int) {
        guard let path = zone.getLink(rel: "self") else { fatalError() }
        guard let url = URL(string: "\(self.url)\(path)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "PATCH"
        if let nextWaterTime = zone.nextWaterTime {
            let requestBody = DelayWateringRequestBody(waterSchedule: WaterScheduleStartTime(startTime: Calendar.current.date(byAdding: .day, value: days, to: nextWaterTime)!))
            let jsonData = try! DelayWateringRequestBody.encoder.encode(requestBody)
//            print("DATA: \(String(data: jsonData, encoding: String.Encoding.utf8))")
            request.httpBody = jsonData
            let task = URLSession.shared.dataTask(with: request)
            task.resume()
        }
    }

    func endDateZone(zone: Zone) {
        guard let path = zone.getLink(rel: "self") else { fatalError() }
        guard let url = URL(string: "\(self.url)\(path)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        let task = URLSession.shared.dataTask(with: request)
        task.resume()
    }
    
    func restoreZone(zone: Zone) {
        guard let path = zone.getLink(rel: "self") else { fatalError() }
        guard let url = URL(string: "\(self.url)\(path)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "PATCH"
        let data: Dictionary<String, Date?> = ["end_date": nil]
        let jsonData = try! JSONEncoder().encode(data)
        request.httpBody = jsonData
        let task = URLSession.shared.dataTask(with: request)
        task.resume()
    }

    func updateZone(zone: Zone, newZone: UpdateZoneRequest) {
        guard let path = zone.getLink(rel: "self") else { fatalError() }
        guard let url = URL(string: "\(self.url)\(path)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "PATCH"
        let jsonData = try! Zone.encoder.encode(newZone)
        request.httpBody = jsonData
//        print("DATA: \(String(data: jsonData, encoding: String.Encoding.utf8))")
        let task = URLSession.shared.dataTask(with: request)
        task.resume()
    }
    
    func createZone(garden: Garden, zone: CreateZoneRequest) {
        guard let path = garden.getLink(rel: "zones") else { fatalError() }
        guard let url = URL(string: "\(self.url)\(path)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        let jsonData = try! CreateZoneRequest.encoder.encode(zone)
        request.httpBody = jsonData
//        print("DATA: \(String(data: jsonData, encoding: String.Encoding.utf8))")
        let task = URLSession.shared.dataTask(with: request)
        task.resume()
    }
    
    func getWateringHistory(zone: Zone, range: String = "168h", limit: Int = 5, withCompletion completion: @escaping (WaterHistoryResponse?) -> Void) {
        guard let path = zone.getLink(rel: "history") else { fatalError() }
        guard let url = URL(string: "\(self.url)\(path)?limit=\(limit)&range=\(range)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        let task = URLSession.shared.dataTask(with: request) { (data, response, error) in
            if let data = data {
                do {
                    let history = try WaterHistory.decoder.decode(WaterHistoryResponse.self, from: data)
                    DispatchQueue.main.async { completion(history) }
                } catch let jsonErr {
                    print("JSON Error: \(jsonErr)")
                    DispatchQueue.main.async { completion(nil) }
                }
            }
        }
        task.resume()
    }
}
