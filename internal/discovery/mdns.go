package discovery

const ServiceType = "_bitexchange._tcp"

func ServiceInstanceName(deviceID string) string {
    return "bitexchange-" + deviceID
}
