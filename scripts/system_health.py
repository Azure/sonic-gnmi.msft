import json
import sys

def collect_health(output_file):
    try:
        from health_checker.manager import HealthCheckerManager
        from sonic_platform.chassis import Chassis

        manager = HealthCheckerManager()
        if not manager.config.config_file_exists():
            result = {"error": "config_missing"}
        else:
            chassis = Chassis()
            stat = manager.check(chassis)
            chassis.initizalize_system_led()
            led = chassis.get_status_led()
            ignore_services = list(manager.config.ignore_services) if manager.config.ignore_services else []
            ignore_devices = list(manager.config.ignore_devices) if manager.config.ignore_devices else []
            result = {"led": led, "stat": stat, "ignore_services": ignore_services, "ignore_devices": ignore_devices}

        with open(output_file, 'w') as f:
            json.dump(result, f)
    except Exception as e:
        with open(output_file, 'w') as f:
            json.dump({"error": str(e)}, f)

def run(output_file):
    collect_health(output_file)

if __name__ == "__main__":
    run(sys.argv[1])