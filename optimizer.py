import os
import platform
import subprocess
import sys

from mystic.monitors import VerboseMonitor
from mystic.solvers import *
from mystic.strategy import *
from mystic.termination import *

CALLS_BETWEEN_IMAGE = 0
MAX_VELOCITY_ALLOWED = 30.0
MIN_VELOCITY_ALLOWED = 1.0
MAX_ACCELERATION_ALLOWED = 3.0  # m/s^2
MAX_DECCELERATION_ALLOWED = -2.0
MAX_ENERGY_CONS = 1300
MAX_CENTRIPETAL_ALLOWED = 3.0  # m/s^2

cli_program = "./asc-simulation.exe"

args = sys.argv

args.pop(0)

cli_cmd = [cli_program] + ["calc"] + list(map(str, args))
print(" ".join(map(str, test)))


def call_cli_program(x):
    cli_cmd[4] = x[0]
    cli_cmd[6] = x[1]
    return subprocess.run(
        cli_cmd,
        capture_output=True,
        text=True,
    ).stdout


output_cache = {}
i = 0


def get_output(x):
    x_tuple = tuple(x)
    if x_tuple not in output_cache:
        output_cache[x_tuple] = call_cli_program(x)
    return output_cache[x_tuple]


def parse_value(value, output):
    return float(output.split(value)[1].split("\n")[0])


def objective(strategy_to_test):
    output = get_output(strategy_to_test)
    time_elapsed = parse_value("Time Elapsed (s):", output)
    energy_consumption = parse_value("Energy Consumption (W):", output)
    initial_velocity = parse_value("Initial Velocity (m/s):", output)
    final_velocity = parse_value("Final Velocity (m/s):", output)
    max_velocity = parse_value("Max Velocity (m/s):", output)
    min_velocity = parse_value("Min Velocity (m/s):", output)
    max_acceleration = parse_value("Max Acceleration (m/s^2):", output)
    min_acceleration = parse_value("Min Acceleration (m/s^2):", output)
    max_centripetal_force = parse_value("Max Centripetal Acceleration (m/s^2):", output)

    objective_value = abs(time_elapsed)

    # Check energy consumption constraint
    if energy_consumption > MAX_ENERGY_CONS:
        objective_value += abs(energy_consumption - MAX_ENERGY_CONS) * 100000

    if energy_consumption < 0:
        objective_value += abs(energy_consumption) * 100000

    if max_velocity > MAX_VELOCITY_ALLOWED:
        objective_value += abs(max_velocity - MAX_VELOCITY_ALLOWED) * 100000

    if min_velocity < MIN_VELOCITY_ALLOWED:
        objective_value += abs(min_velocity) * 100000

    if max_acceleration > MAX_ACCELERATION_ALLOWED:
        objective_value += abs(MAX_ACCELERATION_ALLOWED - max_acceleration) * 100000

    if min_acceleration < MAX_DECCELERATION_ALLOWED:
        objective_value += abs(min_acceleration - MAX_DECCELERATION_ALLOWED) * 100000

    if max_centripetal_force > MAX_CENTRIPETAL_ALLOWED:
        objective_value += abs(max_centripetal_force - MAX_CENTRIPETAL_ALLOWED) * 100000

    velocity_difference = abs(final_velocity - initial_velocity)
    objective_value += velocity_difference * 100000

    return (
        objective_value
        if objective_value != float("inf") and objective_value >= 0
        else sys.float_info.max
    )


# Initialization
mon = VerboseMonitor(10, 50)

lower = [0.0, 0]
upper = [65.0, 5]

# Configure and solve using LatticeSolver
solver = BuckshotSolver(2)
solver.SetGenerationMonitor(mon)
solver.SetStrictRanges(lower, upper)
solver.SetEvaluationLimits(10000000, 10000000)
solver.SetTermination(SolverInterrupt())

try:
    solver.Solve(objective, disp=True)
except KeyboardInterrupt:
    print("\nOptimization interrupted by user.\n")

res = solver.Solution()
print("Optimized Result:", res)
print("Objective Value:", objective(res))
output_cache.clear()
print(call_cli_program(res, ""))
