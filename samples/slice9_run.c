float addf(float a, float b) {
    return a + b;
}

int main() {
    float x = 2.5;
    float y = 4.0;
    float z = addf(x, y);
    print_float(z);
    print_newline();
    return 0;
}
