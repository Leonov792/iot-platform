import SwiftUI
import Charts

// MARK: - Модели

struct DeviceState: Codable {
    var on: Bool?
    var brightness: Double?
    var targetTemp: Double?

    enum CodingKeys: String, CodingKey {
        case on, brightness
        case targetTemp = "target_temp"
    }
}

struct Device: Codable, Identifiable {
    let id: String
    let name: String
    let type: String
    let status: String
    let room: String
    let state: DeviceState?
    let ownerId: String?
    let createdAt: String?
    let lastSeen: String?

    enum CodingKeys: String, CodingKey {
        case id, name, type, status, room, state
        case ownerId = "owner_id"
        case createdAt = "created_at"
        case lastSeen = "last_seen"
    }

    var isOn: Bool { state?.on ?? false }
}

struct Payload: Codable {
    let temp: Double?
    let humidity: Double?
    let battery: Double?
}

struct Telemetry: Codable {
    let id: Int
    let deviceId: String
    let ts: String
    let payload: Payload

    enum CodingKeys: String, CodingKey {
        case id, ts, payload
        case deviceId = "device_id"
    }
}

struct LivePoint: Codable {
    let deviceId: String
    let ts: Double
    let temp: Double
    let humidity: Double
    let battery: Double

    enum CodingKeys: String, CodingKey {
        case deviceId = "device_id"
        case ts, temp, humidity, battery
    }
}

struct AuthResponse: Codable { let token: String }
struct ErrorResponse: Codable { let error: String }

// MARK: - API

final class APIClient: ObservableObject {
    static let shared = APIClient()
    let base = "http://localhost:8080"

    @Published var token: String? = UserDefaults.standard.string(forKey: "token")

    func setToken(_ t: String?) {
        token = t
        UserDefaults.standard.set(t, forKey: "token")
    }

    private func request(_ method: String, _ path: String, body: [String: Any]? = nil) async throws -> Data {
        var req = URLRequest(url: URL(string: base + path)!)
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let token { req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }
        if let body { req.httpBody = try JSONSerialization.data(withJSONObject: body) }

        let (data, resp) = try await URLSession.shared.data(for: req)
        if let http = resp as? HTTPURLResponse, !(200...299).contains(http.statusCode) {
            let msg = (try? JSONDecoder().decode(ErrorResponse.self, from: data))?.error ?? "ошибка \(http.statusCode)"
            throw NSError(domain: "api", code: http.statusCode, userInfo: [NSLocalizedDescriptionKey: msg])
        }
        return data
    }

    func login(email: String, password: String) async throws {
        let data = try await request("POST", "/api/v1/auth/login", body: ["email": email, "password": password])
        setToken(try JSONDecoder().decode(AuthResponse.self, from: data).token)
    }

    func register(email: String, password: String) async throws {
        let data = try await request("POST", "/api/v1/auth/register", body: ["email": email, "password": password])
        setToken(try JSONDecoder().decode(AuthResponse.self, from: data).token)
    }

    func devices() async throws -> [Device] {
        try JSONDecoder().decode([Device].self, from: try await request("GET", "/api/v1/devices"))
    }

    func createDevice(id: String, name: String, type: String, room: String) async throws {
        _ = try await request("POST", "/api/v1/devices", body: ["id": id, "name": name, "type": type, "room": room])
    }

    func deleteDevice(id: String) async throws {
        _ = try await request("DELETE", "/api/v1/devices/\(id)")
    }

    func command(id: String, action: String, value: Double? = nil) async throws {
        var body: [String: Any] = ["action": action]
        if let value { body["value"] = value }
        _ = try await request("POST", "/api/v1/devices/\(id)/command", body: body)
    }

    func telemetry(id: String) async throws -> [Telemetry] {
        try JSONDecoder().decode([Telemetry].self, from: try await request("GET", "/api/v1/devices/\(id)/telemetry"))
    }
}

// MARK: - WebSocket

final class TelemetryStream: ObservableObject {
    @Published var points: [LivePoint] = []
    private var task: URLSessionWebSocketTask?

    func connect(deviceId: String) {
        task = URLSession.shared.webSocketTask(with: URL(string: "ws://localhost:4000/ws/dashboard")!)
        task?.resume()
        task?.send(.string(#"{"type":"subscribe","device_id":"\#(deviceId)"}"#)) { _ in }
        receive()
    }

    private func receive() {
        task?.receive { [weak self] result in
            guard let self else { return }
            switch result {
            case .success(let message):
                if case .string(let text) = message,
                   let data = text.data(using: .utf8),
                   let p = try? JSONDecoder().decode(LivePoint.self, from: data) {
                    DispatchQueue.main.async { self.points.append(p) }
                }
                self.receive()
            case .failure:
                break
            }
        }
    }

    func close() {
        task?.cancel(with: .normalClosure, reason: nil)
    }
}

// MARK: - Корневой вью

struct ContentView: View {
    enum Route { case login, devices, detail(String) }

    @State private var route: Route = APIClient.shared.token == nil ? .login : .devices

    var body: some View {
        switch route {
        case .login:
            LoginView { route = .devices }
        case .devices:
            DevicesView(
                onOpen: { route = .detail($0) },
                onLogout: { APIClient.shared.setToken(nil); route = .login }
            )
        case .detail(let id):
            DeviceDetailView(id: id, onBack: { route = .devices })
        }
    }
}

// MARK: - Логин

struct LoginView: View {
    let onDone: () -> Void
    @State private var email = ""
    @State private var password = ""
    @State private var register = false
    @State private var error: String?
    @State private var loading = false

    var body: some View {
        VStack(spacing: 16) {
            Text("Умный дом").font(.largeTitle).bold()
            Text(register ? "Регистрация" : "Вход").foregroundStyle(.secondary)

            TextField("Почта", text: $email)
                .textContentType(.emailAddress)
                .textFieldStyle(.roundedBorder)
            SecureField("Пароль", text: $password)
                .textFieldStyle(.roundedBorder)

            if let error { Text(error).foregroundStyle(.red).font(.footnote) }

            Button(register ? "Создать аккаунт" : "Войти") {
                loading = true
                Task {
                    do {
                        if register { try await APIClient.shared.register(email: email, password: password) }
                        else { try await APIClient.shared.login(email: email, password: password) }
                        onDone()
                    } catch let err {
                        error = err.localizedDescription
                    }
                    loading = false
                }
            }
            .buttonStyle(.borderedProminent)
            .disabled(loading)

            Button(register ? "Уже есть? Войти" : "Нет аккаунта? Регистрация") {
                register.toggle()
            }
            .font(.footnote)
        }
        .padding()
    }
}

// MARK: - Список устройств

struct DevicesView: View {
    let onOpen: (String) -> Void
    let onLogout: () -> Void

    @State private var devices: [Device] = []
    @State private var loading = true
    @State private var adding = false
    @State private var name = ""
    @State private var type = "light"
    @State private var room = ""
    @State private var id = ""

    private var grouped: [(key: String, value: [Device])] {
        Dictionary(grouping: devices, by: { $0.room.isEmpty ? "Без комнаты" : $0.room })
            .sorted { $0.key < $1.key }
    }

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    ProgressView()
                } else {
                    List {
                        ForEach(grouped, id: \.key) { group in
                            Section(group.key) {
                                ForEach(group.value) { d in
                                    HStack {
                                        VStack(alignment: .leading) {
                                            Text(d.name)
                                            Text("\(typeLabel(d.type)) · \(d.status)")
                                                .font(.caption).foregroundStyle(.secondary)
                                        }
                                        Spacer()
                                        if d.type == "light" || d.type == "plug" {
                                            Button(d.isOn ? "Выкл" : "Вкл") {
                                                Task { try? await APIClient.shared.command(id: d.id, action: d.isOn ? "off" : "on") }
                                            }
                                            .buttonStyle(.bordered)
                                        } else {
                                            Button("Открыть") { onOpen(d.id) }
                                                .buttonStyle(.bordered)
                                        }
                                    }
                                    .contentShape(Rectangle())
                                    .onTapGesture { onOpen(d.id) }
                                }
                                .onDelete { idx in
                                    for i in idx {
                                        let d = group.value[i]
                                        Task { try? await APIClient.shared.deleteDevice(id: d.id); reload() }
                                    }
                                }
                            }
                        }
                    }
                }
            }
            .navigationTitle("Устройства")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Выйти") { onLogout() }
                }
                ToolbarItem(placement: .topBarLeading) {
                    Button("+ Добавить") { adding.toggle() }
                }
            }
            .sheet(isPresented: $adding) {
                Form {
                    TextField("Название", text: $name)
                    TextField("Тип", text: $type)
                    TextField("Комната", text: $room)
                    TextField("ID (lamp-1)", text: $id)
                    Button("Создать") {
                        Task {
                            try? await APIClient.shared.createDevice(id: id, name: name, type: type, room: room)
                            adding = false
                            name = ""; id = ""; room = ""
                            reload()
                        }
                    }
                    .disabled(id.isEmpty || name.isEmpty)
                }
            }
            .task { reload() }
        }
    }

    private func reload() {
        loading = true
        Task {
            devices = (try? await APIClient.shared.devices()) ?? []
            loading = false
        }
    }
}

// MARK: - Детали устройства

struct DeviceDetailView: View {
    let id: String
    let onBack: () -> Void

    @State private var device: Device?
    @State private var history: [Double] = []
    @State private var target = "22"
    @StateObject private var stream = TelemetryStream()

    private var temps: [Double] { history + stream.points.map(\.temp) }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                HStack {
                    Button("← Назад") { onBack() }
                    Spacer()
                    Text(device?.name ?? "...").font(.title2).bold()
                    Spacer()
                }

                if let d = device {
                    Text(d.room).foregroundStyle(.secondary)
                    Text("статус: \(d.status)")
                        .foregroundStyle(d.status == "online" ? Color.green : Color.secondary)
                }

                controls

                Divider()

                Text("Телеметрия").font(.headline)
                if let last = stream.points.last {
                    Text("t \(last.temp, specifier: "%.1f")° · h \(last.humidity, specifier: "%.0f")% · заряд \(last.battery, specifier: "%.0f")%")
                        .font(.caption).foregroundStyle(.secondary)
                }

                if temps.count < 2 {
                    Text("ждём данные...").foregroundStyle(.secondary)
                        .frame(height: 180)
                } else {
                    Chart {
                        ForEach(Array(temps.enumerated()), id: \.offset) { i, v in
                            LineMark(x: .value("t", i), y: .value("°C", v))
                        }
                    }
                    .frame(height: 180)
                }
            }
            .padding()
        }
        .task {
            let list = (try? await APIClient.shared.devices()) ?? []
            device = list.first { $0.id == id }
            history = ((try? await APIClient.shared.telemetry(id: id)) ?? []).compactMap(\.payload.temp)
            stream.connect(deviceId: id)
        }
        .onDisappear { stream.close() }
    }

    @ViewBuilder
    private var controls: some View {
        switch device?.type {
        case "light":
            HStack {
                Button(device?.isOn == true ? "Выключить" : "Включить") {
                    Task {
                        try? await APIClient.shared.command(id: id, action: device?.isOn == true ? "off" : "on")
                        await refresh()
                    }
                }
                .buttonStyle(.borderedProminent)
            }
        case "plug":
            Button(device?.isOn == true ? "Выключить" : "Включить") {
                Task {
                    try? await APIClient.shared.command(id: id, action: device?.isOn == true ? "off" : "on")
                    await refresh()
                }
            }
            .buttonStyle(.borderedProminent)
        case "thermostat":
            HStack {
                TextField("°C", text: $target).frame(width: 80).textFieldStyle(.roundedBorder)
                Button("Применить") {
                    if let v = Double(target) {
                        Task { try? await APIClient.shared.command(id: id, action: "set_target", value: v) }
                    }
                }
                .buttonStyle(.borderedProminent)
            }
        default:
            EmptyView()
        }
    }

    private func refresh() async {
        let list = (try? await APIClient.shared.devices()) ?? []
        device = list.first { $0.id == id }
    }
}

private func typeLabel(_ type: String) -> String {
    switch type {
    case "light": return "Лампа"
    case "plug": return "Розетка"
    case "thermostat": return "Термостат"
    case "sensor": return "Датчик"
    default: return type
    }
}
