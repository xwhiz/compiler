int main() {
    int nums[4];
    nums[0] = 1;
    nums[1] = 2;
    nums[2] = 3;
    nums[3] = nums[0] + nums[1] + nums[2];
    print_int(nums[3]);
    print_newline();
    return nums[3];
}
