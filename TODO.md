### **Core Concepts to Learn**  
#### **Digital Biology Basics**  
- **Artificial Life**: Study Conway’s *Game of Life*, *Tierra*, and *Avida* (digital evolution simulators).  
- **Genetic Algorithms (GA)**:  
  - How "organisms" (goroutines) encode traits (e.g., CPU usage, replication rate).  
  - *Fitness functions*: What determines "survival" (e.g., resource efficiency).  
- **Mutation/Selection**: Random changes + competition for resources (CPU, memory).  

---

### **Technical Dependencies to Explore**   
- **Math/Stats**:  
  - Probability distributions (for mutation rates).  
  - Evolutionary stable strategies (ESS) for balancing traits.  
---

### **Inspiration & Further Reading**  
- **Books**:  
  - *"The Selfish Gene"* (Dawkins) – Evolutionary biology basics.  
  - *"Complexity: A Guided Tour"* (Mitchell) – Emergent systems.  
- **Papers**:  
  - [*"Avida: A Software Platform for Research in Computational Evolutionary Biology"*](https://doi.org/10.1038/s41596-020-0293-9).  
- **Videos**:  
  - [*"Digital Evolution: Creating Artificial Life"* (Youtube)](https://www.youtube.com/watch?v=oCXzcPNsqGA).  
---

### **Future Considerations**
- Right now once the organism acquires CPU tokens it is entirely up to that organism to release it. Future versions may change that.
- Explore statistical properties of the random mutation functions in `world/utils.go`
