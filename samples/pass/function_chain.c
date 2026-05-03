int double_it(int x) {
    return x + x;
}

int inc(int x) {
    return x + 1;
}

int main() {
    int y = inc(double_it(5));
    print_int(y);
    print_newline();
    return y;
}
