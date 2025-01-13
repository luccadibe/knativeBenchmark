from parliament import Context

is_cold = True

def main(context: Context):
    global is_cold
    response = str(is_cold)
    is_cold = False
    return response, 200
