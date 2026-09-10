# list_ops.py - 列表、下标、赋值

nums = [3, 1, 4, 1, 5, 9, 2, 6]
print("len(nums) =", len(nums))
print("nums[0] =", nums[0])
print("nums[-1] =", nums[-1])

# 下标赋值
nums[0] = 100
print("after nums[0]=100:", nums)

# 求和
total = 0
for x in nums:
    total = total + x
print("sum =", total)

# 找最大
m = nums[0]
for x in nums:
    if x > m:
        m = x
print("max =", m)